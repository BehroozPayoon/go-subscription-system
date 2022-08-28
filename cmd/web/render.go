package main

import (
	"fmt"
	"html/template"
	"net/http"
	"subscription-system/data"
	"time"
)

var pathToTemplates = "./cmd/web/templates"

type TemplateData struct {
	StringMap     map[string]string
	IntMap        map[string]int
	FloatMap      map[string]float64
	Data          map[string]any
	Flash         string
	Warning       string
	Error         string
	Authenticated bool
	Now           time.Time
	User          *data.User
}

func (app *Config) render(w http.ResponseWriter, req *http.Request, t string, td *TemplateData) {
	partials := []string{
		fmt.Sprintf("%s/base.layout.gohtml", pathToTemplates),
		fmt.Sprintf("%s/header.partial.gohtml", pathToTemplates),
		fmt.Sprintf("%s/navbar.partial.gohtml", pathToTemplates),
		fmt.Sprintf("%s/footer.partial.gohtml", pathToTemplates),
		fmt.Sprintf("%s/alerts.partial.gohtml", pathToTemplates),
	}
	var templateSlice []string
	templateSlice = append(templateSlice, fmt.Sprintf("%s/%s", pathToTemplates, t))

	for _, x := range partials {
		templateSlice = append(templateSlice, x)
	}

	if td == nil {
		td = &TemplateData{}
	}

	tmpl, err := template.ParseFiles(templateSlice...)
	if err != nil {
		app.ErrorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, app.AddDefaultData(td, req)); err != nil {
		app.ErrorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (app *Config) AddDefaultData(td *TemplateData, req *http.Request) *TemplateData {
	td.Flash = app.Session.PopString(req.Context(), "flash")
	td.Warning = app.Session.PopString(req.Context(), "warning")
	td.Error = app.Session.PopString(req.Context(), "error")
	if app.IsAuthenticated(req) {
		td.Authenticated = true
		user, ok := app.Session.Get(req.Context(), "user").(data.User)
		if !ok {
			app.ErrorLog.Println("Can't get user from session")
		} else {
			td.User = &user
		}
	}
	td.Now = time.Now()

	return td
}

func (app *Config) IsAuthenticated(req *http.Request) bool {
	return app.Session.Exists(req.Context(), "userID")
}
