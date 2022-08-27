package main

import "net/http"

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

}

func (app *Config) Activate(w http.ResponseWriter, req *http.Request) {

}
