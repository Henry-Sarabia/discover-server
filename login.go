package main

import (
	"github.com/go-chi/render"
	"net/http"
)

// Login contains the URL configured for Spotify authentication.
type Login struct {
	URL string `json:"url"`
}

// loginHandler responds to requests with an authorization URL configured for a
// user's Spotify data. In addition, a session is created to store the caller's
// UUID and time of request. Sessions are saved as secure, encrypted cookies.
func loginHandler(w http.ResponseWriter, r *http.Request) {
	url := auth.AuthURL(state)

	login := Login{URL: url}
	render.JSON(w, r, login)
}
