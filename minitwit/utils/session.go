package utils

import (
	"net/http"

	"github.com/gorilla/sessions"
)

const SECRET_KEY = "development key"

var sessionSaveError = "Failed to save session"
var sessionGetError = "Failed to get session"

var store *sessions.CookieStore

func init() {

	store = sessions.NewCookieStore([]byte(SECRET_KEY))

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// Flash messages
func AddFlash(w http.ResponseWriter, r *http.Request, message string) {
	session, err := GetSession(r, w)
	if err != nil {
		http.Error(w, sessionGetError, http.StatusInternalServerError)
		return
	}
	session.AddFlash(message)
	if err := session.Save(r, w); err != nil {
		http.Error(w, sessionSaveError, http.StatusInternalServerError)
	}
}

func GetFlashes(w http.ResponseWriter, r *http.Request) []interface{} {
	session, err := GetSession(r, w)
	if err != nil {
		http.Error(w, sessionGetError, http.StatusInternalServerError)
		return nil
	}
	flashes := session.Flashes()
	if err := session.Save(r, w); err != nil {
		http.Error(w, sessionSaveError, http.StatusInternalServerError)
		return nil
	}
	return flashes
}

// Get session
func GetSession(r *http.Request, w http.ResponseWriter) (*sessions.Session, error) {
	session, err := store.Get(r, "minitwit-session")
	if err != nil {
		// Handle invalid cookie case
		session.Options.MaxAge = -1
		if err := session.Save(r, w); err != nil {
			http.Error(w, sessionSaveError, http.StatusInternalServerError)
		}
		http.Redirect(w, r, "/login", 419)
	}
	return session, err
}
