package main

import "net/http"

func (app *Config) SessionLoad(next http.Handler) http.Handler {
	return app.Session.LoadAndSave(next)
}

func (app *Config) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !app.Session.Exists(req.Context(), "userID") {
			app.Session.Put(req.Context(), "error", "Log in first")
			http.Redirect(w, req, "/login", http.StatusTemporaryRedirect)
			return
		}
		next.ServeHTTP(w, req)
	})
}
