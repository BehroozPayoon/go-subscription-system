package main

import (
	"fmt"
	"github.com/phpdave11/gofpdf"
	"github.com/phpdave11/gofpdf/contrib/gofpdi"
	"html/template"
	"net/http"
	"strconv"
	"subscription-system/data"
	"time"
)

func (app *Config) HomePage(w http.ResponseWriter, req *http.Request) {
	app.render(w, req, "home.page.gohtml", nil)
}

func (app *Config) LoginPage(w http.ResponseWriter, req *http.Request) {
	app.render(w, req, "login.page.gohtml", nil)
}

func (app *Config) PostLoginPage(w http.ResponseWriter, req *http.Request) {
	_ = app.Session.RenewToken(req.Context())

	err := req.ParseForm()
	if err != nil {
		app.ErrorLog.Println(err)
	}

	email := req.Form.Get("email")
	password := req.Form.Get("password")

	user, err := app.Models.User.GetByEmail(email)
	if err != nil {
		app.Session.Put(req.Context(), "error", "Invalid credentials.")
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}

	validPassword, err := user.PasswordMatches(password)
	if err != nil {
		app.Session.Put(req.Context(), "error", "Invalid credentials.")
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}

	if !validPassword {
		msg := Message{
			To:      email,
			Subject: "Failed log in attempt",
			Data:    "Invalid login attempt",
		}
		app.sendEmail(msg)

		app.Session.Put(req.Context(), "error", "Invalid credentials.")
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}

	app.Session.Put(req.Context(), "userID", user.ID)
	app.Session.Put(req.Context(), "user", user)

	app.Session.Put(req.Context(), "flash", "Successful login!")
	http.Redirect(w, req, "/", http.StatusSeeOther)
}

func (app *Config) Logout(w http.ResponseWriter, req *http.Request) {
	_ = app.Session.Destroy(req.Context())
	_ = app.Session.RenewToken(req.Context())
	http.Redirect(w, req, "/login", http.StatusSeeOther)
}

func (app *Config) RegisterPage(w http.ResponseWriter, req *http.Request) {
	app.render(w, req, "register.page.gohtml", nil)
}

func (app *Config) PostRegisterPage(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		app.ErrorLog.Println(err)
	}
	u := data.User{
		Email:     req.Form.Get("email"),
		FirstName: req.Form.Get("first_name"),
		LastName:  req.Form.Get("last_name"),
		Password:  req.Form.Get("Password"),
		Active:    0,
		IsAdmin:   0,
	}

	_, err = u.Insert(u)
	if err != nil {
		app.Session.Put(req.Context(), "error", "unable to create user")
		http.Redirect(w, req, "/register", http.StatusSeeOther)
		return
	}

	url := fmt.Sprintf("https://payoon.dev/activate?email=%s", u.Email)
	signedURL := GenerateTokenFromString(url)

	msg := Message{
		To:       u.Email,
		Subject:  "Activate your account",
		Template: "confirmation-email",
		Data:     template.HTML(signedURL),
	}
	app.sendEmail(msg)

	app.Session.Put(req.Context(), "flash", "Confirmation email sent. Check your email")
	http.Redirect(w, req, "/login", http.StatusSeeOther)
}

func (app *Config) Activate(w http.ResponseWriter, req *http.Request) {
	url := req.RequestURI
	testURL := fmt.Sprintf("https://payoon%s", url)
	okay := VerifyToken(testURL)

	if !okay {
		app.Session.Put(req.Context(), "error", "Invalid Token")
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}

	u, err := app.Models.User.GetByEmail(req.URL.Query().Get("email"))
	if err != nil {
		app.Session.Put(req.Context(), "error", "No user found")
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}

	u.Active = 1
	err = u.Update()
	if err != nil {
		app.Session.Put(req.Context(), "error", "Unable to update user")
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}

	app.Session.Put(req.Context(), "flash", "Account activate")
	http.Redirect(w, req, "/login", http.StatusSeeOther)
}

func (app *Config) ChooseSubscription(w http.ResponseWriter, req *http.Request) {
	plans, err := app.Models.Plan.GetAll()
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}

	dataMap := make(map[string]any)
	dataMap["plans"] = plans

	app.render(w, req, "plans.page.gohtml", &TemplateData{
		Data: dataMap,
	})
}

func (app *Config) SubscribeToPlan(w http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")

	planId, err := strconv.Atoi(id)
	if err != nil {
		app.ErrorLog.Println("Error getting planid:", err)
	}

	plan, err := app.Models.Plan.GetOne(planId)
	if err != nil {
		app.Session.Put(req.Context(), "error", "Unable to find plan")
		http.Redirect(w, req, "/members/plans", http.StatusSeeOther)
		return
	}

	user, ok := app.Session.Get(req.Context(), "user").(data.User)
	if !ok {
		app.Session.Put(req.Context(), "error", "login first")
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}

	app.Wait.Add(1)
	func() {
		defer app.Wait.Done()

		invoice, err := app.getInvoice(user, plan)
		if err != nil {
			app.ErrorChan <- err
		}

		msg := Message{
			To:       user.Email,
			Subject:  "Your Invoice",
			Data:     invoice,
			Template: "invoice",
		}
		app.sendEmail(msg)
	}()

	app.Wait.Add(1)
	go func() {
		defer app.Wait.Done()

		pdf := app.generateManual(user, plan)
		err := pdf.OutputFileAndClose(fmt.Sprintf("./tmp/%d_manual.pdf", user.ID))
		if err != nil {
			app.ErrorChan <- err
		}

		msg := Message{
			To:       user.Email,
			Subject:  "Your Manuals",
			Data:     "Your user manual is attached",
			Template: "invoice",
			AttachmentsMap: map[string]string{
				"Manual.pdf": fmt.Sprintf("./tmp/%d_manual.pdf", user.ID),
			},
		}
		app.sendEmail(msg)
	}()

	err = app.Models.Plan.SubscribeUserToPlan(user, *plan)
	if err != nil {
		app.Session.Put(req.Context(), "error", "Error subscribing plan")
		http.Redirect(w, req, "/members/plans", http.StatusSeeOther)
		return
	}

	u, err := app.Models.User.GetOne(user.ID)
	if err != nil {
		app.Session.Put(req.Context(), "error", "Error getting user")
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}
	app.Session.Put(req.Context(), "user", u)

	app.Session.Put(req.Context(), "flash", "Subscribed!")
	http.Redirect(w, req, "/members/plans", http.StatusSeeOther)
}

func (app *Config) getInvoice(user data.User, plan *data.Plan) (string, error) {
	return plan.PlanAmountFormatted, nil
}

func (app *Config) generateManual(user data.User, plan *data.Plan) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(10, 13, 10)

	importer := gofpdi.NewImporter()
	time.Sleep(5 * time.Second)

	t := importer.ImportPage(pdf, "./pdf/manual", 1, "/MediaBox")
	pdf.AddPage()

	importer.UseImportedTemplate(pdf, t, 0, 0, 215.9, 0)

	pdf.SetX(75)
	pdf.SetY(150)

	pdf.SetFont("Arial", "", 12)
	pdf.MultiCell(0, 4, fmt.Sprintf("%s %s", user.FirstName, user.LastName), "", "C", false)
	pdf.Ln(5)
	pdf.MultiCell(0, 4, fmt.Sprintf("%s User Guide", plan.PlanName), "", "C", false)

	return pdf
}
