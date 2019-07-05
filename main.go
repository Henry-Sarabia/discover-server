package main

import (
	"encoding/hex"
	spotifyservice "github.com/Henry-Sarabia/refind/spotify"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/zmb3/spotify"
)

const (
	redirectPath    string = "/results"
	state           string = "abc123"
	hashKeyName     string = "DISCOVER_HASH"
	storeAuthName   string = "DISCOVER_AUTH"
	storeCryptName  string = "DISCOVER_CRYPT"
	frontendURIName string = "FRONTEND_URI"
)

var (
	hashKey     string
	storeAuth   string
	storeCrypt  string
	store       sessions.CookieStore
	frontendURI string
	auth        *spotify.Authenticator
)

func init() {
	var err error

	hashKey, err = decodeEnv(hashKeyName)
	if err != nil {
		log.Fatal(err)
	}

	storeAuth, err = decodeEnv(storeAuthName)
	if err != nil {
		log.Fatal(err)
	}

	storeCrypt, err = decodeEnv(storeCryptName)
	if err != nil {
		log.Fatal(err)
	}

	frontendURI, err = getEnv(frontendURIName)
	if err != nil {
		log.Fatal(err)
	}

	auth, err = spotifyservice.Authenticator(frontendURI + redirectPath)
	if err != nil {
		log.Fatalf("stack trace:\n%+v\n", err)
	}
}

func decodeEnv(name string) (string, error) {
	env, err := getEnv(name)
	if err != nil {
		return "", err
	}

	d, err := hex.DecodeString(env)
	if err != nil {
		return "", errors.Wrapf(err, "environment variable with name '%s' cannot be decoded from hex", name)
	}

	return string(d), nil
}

func getEnv(name string) (string, error) {
	env, ok := os.LookupEnv(name)
	if !ok {
		return "", errors.Errorf("environment variable with name '%s' cannot be found", name)
	}

	return env, nil
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
