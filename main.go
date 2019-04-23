package main

import (
	"fmt"
	"github.com/Henry-Sarabia/refind"
	"github.com/Henry-Sarabia/refind/buffer"
	spotifyservice "github.com/Henry-Sarabia/refind/spotify"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/render"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

const (
	frontendURI   string = "http://127.0.0.1:3000"
	state         string = "abc123"
	playlistLimit int    = 30
)

var (
	redirectURI = frontendURI + "/results"
)

var auth *spotify.Authenticator

func init() {
	fmt.Println(frontendURI)
	var err error

	auth, err = spotifyservice.Authenticator(redirectURI)
	if err != nil {
		log.Printf("stack trace:\n%+v\n", err)
		os.Exit(1)
	}
}

// Login contains the URL configured for Spotify authentication.
type Login struct {
	URL string `json:"url"`
}

// Playlist contains the URI of a user's playlist.
type Playlist struct {
	URI string `json:"uri"`
}

func main() {
	r := mux.NewRouter()

	cors := handlers.CORS(
		handlers.AllowCredentials(),
		handlers.AllowedOrigins([]string{frontendURI}),
		handlers.AllowedMethods([]string{"GET"}),
		handlers.MaxAge(600),
	)
	r.Use(cors)
	r.Use(handlers.RecoveryHandler())

	api := r.PathPrefix("/api/v1/").Subrouter()
	api.HandleFunc("/login", loginHandler)
	api.Handle("/playlist", errHandler(playlistHandler))

	srv := &http.Server{
		Handler:      handlers.LoggingHandler(os.Stdout, r),
		Addr:         "127.0.0.1:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

type serverError struct {
	Error   error
	Message string
	Code    int
}

type errHandler func(http.ResponseWriter, *http.Request) *serverError

func (fn errHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w, r); err != nil {
		http.Error(w, err.Message, err.Code)
	}
}

// loginHandler responds to requests with an authorization URL configured for a
// user's Spotify data. In addition, a session is created to store the caller's
// UUID and time of request. Sessions are saved as secure, encrypted cookies.
func loginHandler(w http.ResponseWriter, r *http.Request) {
	url := auth.AuthURL(state)

	login := Login{URL: url}
	render.JSON(w, r, login)
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

	pl, err := serv.Playlist("Discover Now", list)
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
