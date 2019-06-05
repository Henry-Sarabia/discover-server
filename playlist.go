package main

import (
	"github.com/Henry-Sarabia/refind"
	"github.com/Henry-Sarabia/refind/buffer"
	"github.com/go-chi/render"
	"golang.org/x/oauth2"
	"net/http"

	adj "github.com/nii236/adjectiveadjectiveanimal"
	spotifyservice "github.com/Henry-Sarabia/refind/spotify"
)

// Playlist contains the URI of a user's playlist.
type Playlist struct {
	URI string `json:"uri"`
}

// playlistHandler responds to requests with a Spotify playlist URI generated
// using the authenticated user's playback data. This URI is stored in the
// user's session and is used as the response to any further requests unless
// the URI is cleared from the session.
func playlistHandler(w http.ResponseWriter, r *http.Request) *serverError {
	tok, err := authorizeRequest(w, r)
	if err != nil {
		return &serverError{
			Error:   err,
			Message: "Cannot authorize Spotify request",
			Code:    http.StatusBadGateway,
		}
	}

	c := auth.NewClient(tok)
	c.AutoRetry = true

	serv, err := spotifyservice.New(&c)
	if err != nil {
		return &serverError{
			Error:   err,
			Message: "Something went wrong while initializing Spotify service",
			Code:    http.StatusInternalServerError,
		}
	}

	buf, err := buffer.New(serv)
	if err != nil {
		return &serverError{
			Error:   err,
			Message: "Something went wrong while initializing service buffer",
			Code:    http.StatusInternalServerError,
		}
	}

	gen, err := refind.New(buf, serv)
	if err != nil {
		return &serverError{
			Error:   err,
			Message: "Something went wrong while initializing the refind client",
			Code:    http.StatusInternalServerError,
		}
	}

	list, err := gen.Tracklist(playlistLimit)
	if err != nil {
		return &serverError{
			Error:   err,
			Message: "Something went wrong while generating track list",
			Code:    http.StatusInternalServerError,
		}
	}

	pl, err := serv.Playlist("Discover Now: " + adj.GenerateCombined(1, "-"), list)
	if err != nil {
		return &serverError{
			Error:   err,
			Message: "Something went wrong while creating the user's playlist",
			Code:    http.StatusInternalServerError,
		}
	}

	payload := Playlist{URI: string(pl.URI)}
	render.JSON(w, r, payload)

	return nil
}

// authorizeRequest returns an oauth2 token authenticated for access to a
// particular user's Spotify data after verifying the same user both
// initiated and authorized the request. This verification is done by checking
// for a matching state from the initial request and this subsequent callback.
func authorizeRequest(w http.ResponseWriter, r *http.Request) (*oauth2.Token, error) {
	tok, err := auth.Token(state, r)
	if err != nil {
		return nil, err
	}

	return tok, nil
}
