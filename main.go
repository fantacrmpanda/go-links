package main

import (
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
)

type UserInfo struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Profile       string `json:"profile"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Gender        string `json:"gender"`
}

type link struct {
	Keyword string
	Url     string
	Owner   string
}

type PageData struct {
	Name     string
	Editlink link
	Links    []link
	Errors   string
}

//go:embed assets/*
var assets embed.FS

var sessionManager *scs.SessionManager

func removeAllWhitespaces(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), "")
}

func main() {

	port := flag.String("port", os.Getenv("PORT"), "HTTP port to be used by the server.  Default value is the PORT enviroment variable.")

	if len(*port) == 0 {
		log.Println("No port specified so defaulting to 8080.")
		*port = "8080"
	}
	sessionManager = scs.New()
	sessionManager.Store = sqlite3store.New(db)

	setupGoogleOAuth()
	http.HandleFunc("GET /logout", LogoutHandler)
	http.HandleFunc("POST /edit", SaveHandler)
	http.HandleFunc("GET /edit", DefaultPageHandler)
	http.HandleFunc("/delete/{keyword}", DeleteHandler)
	http.HandleFunc("/{keyword}/{params...}", GetHandler)
	http.HandleFunc("/", DefaultPageHandler)

	log.Printf("Starting HTTP server on %v", *port)
	http.ListenAndServe(":"+*port, sessionManager.LoadAndSave(Authenticate(http.DefaultServeMux)))
}

func init() {
	if _, err := openDatabase("data.db"); err != nil {
		panic(err)
	}

	sessionManager = scs.New()
}

// DefaultPageHandler is responsible for serving the main Go Links
// page.  It will pre-populate the edit form and show the list of
// existing go links.
func DefaultPageHandler(w http.ResponseWriter, r *http.Request) {

	content, err := assets.ReadFile("assets/index.tmpl")
	if err != nil {
		fmt.Fprintf(w, "Error reading file: %v", err)
		return
	}

	tmpl, err := template.New("").Parse(string(content[:]))
	if err != nil {
		fmt.Fprintf(w, "Unable to render page. %v", err)
		return
	}

	// Get any error messages that's been added to the request.
	emsg := r.PostForm.Get("errmsg")

	links, err := allLinks()
	if err != nil {
		log.Printf("error %f\n", err)
		emsg = fmt.Sprintf("%v. %v", emsg, err)
	}

	el := link{
		Keyword: r.URL.Query().Get("keyword"),
		Url:     r.URL.Query().Get("url"),
	}
	pd := PageData{
		Name:     sessionManager.GetString(r.Context(), "email"),
		Editlink: el,
		Links:    links,
		Errors:   emsg,
	}

	if err := tmpl.Execute(w, pd); err != nil {
		fmt.Fprintf(w, "Error executing template %v", err)
	}
}

// GetHandler will do a HTTP 302 redirect to the full URL
// from the given keyword.  If the keyword doesn't exist,
// it will redirect to the main homepage.
func GetHandler(w http.ResponseWriter, r *http.Request) {

	k := r.PathValue("keyword")
	p := r.PathValue("params")
	q := r.URL.RawQuery

	var err error
	var redirect_url string
	if redirect_url, err = getUrlByKeyword(k); err != nil {
		if err == sql.ErrNoRows {
			v := r.URL.Query()
			v.Add("keyword", k)
			r.URL.RawQuery = v.Encode()
			DefaultPageHandler(w, r)
			return
		}
		AddErrorsToRequest(r, err.Error())
		DefaultPageHandler(w, r)
		return
	}

	if len(p) > 0 {
		// Join the params value with the full URL.
		redirect_url, err = url.JoinPath(redirect_url, p)
		if err != nil {
			log.Printf("%s\n", err)
			AddErrorsToRequest(r, err.Error())
			DefaultPageHandler(w, r)
			return
		}
	}

	// Append any query parameters ("?...") before redirecting.
	if len(q) > 0 {
		redirect_url = redirect_url + "?" + q
	}

	http.Redirect(w, r,
		redirect_url,
		http.StatusTemporaryRedirect)

}

// SaveHandler is responsible for creating or updating an Go link.
// Upon saving, it will redirect the user back to the default page.
func SaveHandler(w http.ResponseWriter, r *http.Request) {
	// Starting with Go 1.22, we can tell the router to only send POST
	// method to this handller so we don't have to check the method anymore.

	if err := r.ParseForm(); err != nil {
		AddErrorsToRequest(r, err.Error())
		DefaultPageHandler(w, r)
		return
	}

	k := removeAllWhitespaces(r.Form.Get("keyword"))

	linkurl := r.Form.Get("url")

	if len(linkurl) == 0 || linkurl == "http://" || linkurl == "https://" {
		AddErrorsToRequest(r, "URL can't be blank or just have http prefix.")
		DefaultPageHandler(w, r)
		return
	}
	// If users forgot to put in a http or https prefix then add it for them.
	if !strings.HasPrefix(linkurl, "https://") && !strings.HasPrefix(linkurl, "http://") {
		linkurl = "http://" + linkurl
	}

	// Validate the URL is properly formed.
	if _, err := url.ParseRequestURI(linkurl); err != nil {
		AddErrorsToRequest(r, err.Error())
		DefaultPageHandler(w, r)
		return
	}

	if err := insertOrUpdateLink(k, linkurl, sessionManager.GetString(r.Context(), "email")); err != nil {
		AddErrorsToRequest(r, err.Error())
	}

	DefaultPageHandler(w, r)
}

// DeleteHandler will delete a go link from the database and then redirect
// the user back to the default page.
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	k := r.PathValue("keyword")
	if err := deleteLink(k, sessionManager.GetString(r.Context(), "email")); err != nil {
		AddErrorsToRequest(r, err.Error())
	}
	DefaultPageHandler(w, r)
}

func AddErrorsToRequest(r *http.Request, e string) {
	newmsg := e
	if r.PostForm.Has("errmsg") {
		newmsg = fmt.Sprintf("e. %v", r.PostForm.Get("errmsg"))
		r.PostForm.Del("errmsg")
	}
	r.PostForm.Add("errmsg", newmsg)
}

// Middleware to wrap around handlers to check that user is logged in.
// Usage:
//
//	http.Handle("GET /needauth", Authenticate(homepageHandler))
func Authenticate(h http.Handler) http.Handler {

	authFn := func(w http.ResponseWriter, r *http.Request) {

		email := sessionManager.GetString(r.Context(), "email")

		if len(email) == 0 {
			if strings.HasPrefix(r.URL.Path, "/_callback") {
				googleCallbackHandler(w, r)
				return
			}

			if r.URL.Path == "/auth/login/google" {
				googleHandler(w, r)
				return
			}

			http.Redirect(w, r, "/auth/login/google", http.StatusTemporaryRedirect)
			return
		}

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(authFn)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	sessionManager.Destroy(r.Context())
	fmt.Fprintf(w, "You're logged out.")
}
