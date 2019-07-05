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
	frontendURI   string = "http://127.0.0.1:3000"
	redirectPath  string = "/results"
	state         string = "abc123"
	hashKeyEnv    string = "DISCOVER_HASH"
	storeAuthEnv  string = "DISCOVER_AUTH"
	storeCryptEnv string = "DISCOVER_CRYPT"
)

var (
	auth       *spotify.Authenticator
	hashKey    string
	storeAuth  string
	storeCrypt string
	store      sessions.CookieStore
)

func init() {
	var err error

	auth, err = spotifyservice.Authenticator(frontendURI + redirectPath)
	if err != nil {
		log.Fatalf("stack trace:\n%+v\n", err)
	}

	hashKey, err = decodeEnv(hashKeyEnv)
	if err != nil {
		log.Fatal(err)
	}

	storeAuth, err = decodeEnv(storeAuthEnv)
	if err != nil {
		log.Fatal(err)
	}

	storeCrypt, err = decodeEnv(storeCryptEnv)
	if err != nil {
		log.Fatal(err)
	}
}

func decodeEnv(env string) (string, error) {
	s, ok := os.LookupEnv(env)
	if !ok {
		return "", errors.Errorf("environment variable '%s' is missing", env)
	}

	h, err := hex.DecodeString(s)
	if err != nil {
		return "", errors.Wrapf(err, "environment variable '%s' cannot be decoded from hex", env)
	}

	return string(h), nil
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
