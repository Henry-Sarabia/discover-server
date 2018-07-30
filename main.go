package main

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/render"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"

	uuid "github.com/satori/go.uuid"
)

const (
	sessionName = "discover_now"
)

var (
	frontendURI = os.Getenv("FRONTEND_URI")
	redirectURI = frontendURI + "/results"
	auth        = spotify.NewAuthenticator(
		redirectURI,
		spotify.ScopeUserReadPrivate,
		spotify.ScopeUserTopRead,
		spotify.ScopeUserReadRecentlyPlayed,
		spotify.ScopePlaylistModifyPublic,
	)
	hashKey, hashErr     = hex.DecodeString(os.Getenv("DISCOVER_HASH"))
	storeAuth, authErr   = hex.DecodeString(os.Getenv("DISCOVER_AUTH"))
	storeCrypt, cryptErr = hex.DecodeString(os.Getenv("DISCOVER_CRYPT"))
	store                = sessions.NewCookieStore(storeAuth, storeCrypt)
)

// Login contains the URL configured for Spotify authentication.
type Login struct {
	URL string `json:"url"`
}

// Playlist contains the URI of a user's playlist.
type Playlist struct {
	URI string `json:"uri"`
}

func main() {
	err := verifyEnv()
	if err != nil {
		log.Fatal(err)
	}

	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		// Secure:   false, // true on deploy
		Secure: true,
		MaxAge: 0,
	}

	r := mux.NewRouter()
	r.Use(handlers.RecoveryHandler())

	api := r.PathPrefix("/api/v1/").Subrouter()
	api.HandleFunc("/login", loginHandler)
	api.HandleFunc("/playlist", playlistHandler)

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.PathPrefix("/").HandlerFunc(indexHandler("./index.html"))

	// os.Setenv("PORT", "8080") // remove on deploy
	port, err := getPort()
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Handler: handlers.LoggingHandler(os.Stdout, r),
		// Addr:         "127.0.0.1:" + port,
		Addr:         ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func indexHandler(addr string) func(w http.ResponseWriter, r *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, addr)
	})
}

// loginHandler responds to requests with an authorization URL configured for a
// user's Spotify data. In addition, a session is created to store the caller's
// UUID and time of request. Sessions are saved as secure, encrypted cookies.
func loginHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := store.Get(r, sessionName)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	uid, err := uuid.NewV4()
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id := uid.String()
	time := time.Now().String()

	sess.Values["id"] = id
	sess.Values["time"] = time
	delete(sess.Values, "playlist")
	sess.Save(r, w)

	sum := concatBuf(id, time)
	state, err := hash(sum.Bytes())
	if err != nil {
		log.Println(err)
		http.Error(w, "hash error", http.StatusInternalServerError)
	}

	enc := base64.URLEncoding.EncodeToString(state)
	url := auth.AuthURL(enc)

	login := Login{URL: url}
	render.JSON(w, r, login)
}

// playlistHandler responds to requests with a Spotify playlist URI generated
// using the authenticated user's playback data. This URI is stored in the
// user's session and is used as the response to any further requests unless
// the URI is cleared from the session.
func playlistHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := store.Get(r, sessionName)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if uri, ok := sess.Values["playlist"].(string); ok {
		payload := Playlist{URI: uri}
		render.JSON(w, r, payload)
		return
	}

	tok, err := authorizeRequest(w, r)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nc := auth.NewClient(tok)
	nc.AutoRetry = true

	g := newGenerator(&nc)

	pl, err := g.BestPlaylist()
	if err != nil {
		log.Println(err)
		http.Error(w, "cannot create playlist", http.StatusInternalServerError)
		return
	}

	sess.Values["playlist"] = string(pl.URI)
	sess.Save(r, w)

	payload := Playlist{URI: string(pl.URI)}
	render.JSON(w, r, payload)
}

// authorizeRequest returns an oauth2 token authenticated for access to a
// particular user's Spotify data after verifying the same user both
// initiated and authorized the request. This verification is done by checking
// for a matching state from the initial request and this subsequent callback.
func authorizeRequest(w http.ResponseWriter, r *http.Request) (*oauth2.Token, error) {
	sess, err := store.Get(r, sessionName)
	if err != nil {
		return nil, err
	}

	id, ok := sess.Values["id"].(string)
	if !ok {
		return nil, errors.New("id value not found")
	}

	time, ok := sess.Values["time"].(string)
	if !ok {
		return nil, errors.New("time value not found")
	}

	sum := concatBuf(id, time)
	state, err := hash(sum.Bytes())
	if err != nil {
		return nil, err
	}

	enc := base64.URLEncoding.EncodeToString(state)
	tok, err := auth.Token(enc, r)
	if err != nil {
		return nil, err
	}

	return tok, nil
}

// verifyEnv returns an error if any of the three secret keys are not set
// or cause a decode error.
func verifyEnv() error {
	switch {
	case len(frontendURI) == 0:
		return errors.New("$FRONTEND_URI must be set")
	case hashErr != nil || len(hashKey) == 0:
		return errors.New("$DISCOVER_HASH must be set")
	case authErr != nil || len(storeAuth) == 0:
		return errors.New("$DISCOVER_AUTH must be set")
	case cryptErr != nil || len(storeCrypt) == 0:
		return errors.New("$DISCOVER_CRYPT must be set")
	default:
		return nil
	}
}

// getPort returns the port from the $PORT environment variable as a string.
// Returns an error if $PORT is not set.
func getPort() (string, error) {
	p := os.Getenv("PORT")
	if p == "" {
		return "", errors.New("$PORT must be set")
	}

	return p, nil
}
