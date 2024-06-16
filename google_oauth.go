package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Set up Google OAuth2 config
var gconfig = &oauth2.Config{
	// Replace these values with your actual client ID and secret
	// obtained from https://console.developers.google.com
	ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	RedirectURL:  "http://localhost:8080/_callback", // This should match your configured redirect URI
	Scopes: []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	},
	Endpoint: google.Endpoint,
}

func setupGoogleOAuth() {

	http.HandleFunc("GET /auth/login/google", googleHandler)
	http.HandleFunc("GET /_callback/", googleCallbackHandler)

}

func googleHandler(w http.ResponseWriter, r *http.Request) {

	url := gconfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

}

func googleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")

	// Exchange authorization code for access token.
	t, err := gconfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create oAuth2 client with the token that can be use to call
	// Google's services that fits our scope.
	client := gconfig.Client(context.Background(), t)

	// User the client to fetch the userinfo
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	// Read the response which is a JSON encoded message so unmarshal it to access.
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Redirect(w, r, "/auth/login/google", http.StatusTemporaryRedirect)
	}
	var result UserInfo
	if err := json.Unmarshal(content, &result); err != nil {
		fmt.Printf("error unmarshalling data from google: %v\n", err)
	}

	// var v any
	// // Reading the JSON body using JSON decoder
	//err = json.NewDecoder(resp.Body).Decode(&v)

	sessionManager.Put(r.Context(), "email", result.Email)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)

}
