package main

import "net/http"

func (app *App) ApiHandler(resp http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		resp.Header().Add("Allow", "POST")
		resp.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	hashes := app.translations.Fetch()
	app.refresher.Refresh(hashes)
	resp.WriteHeader(http.StatusAccepted)
}
