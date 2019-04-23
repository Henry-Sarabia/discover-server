package main

import (
	"fmt"
	spotifyservice "github.com/Henry-Sarabia/refind/spotify"
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
