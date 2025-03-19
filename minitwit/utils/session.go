package utils

import (
	"net/http"

	"github.com/gorilla/sessions"
)

const SECRET_KEY = "development key"

var store = sessions.NewCookieStore([]byte(SECRET_KEY))

// Flash messages
func AddFlash(w http.ResponseWriter, r *http.Request, message string) {
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}
	session.AddFlash(message)
	if err := session.Save(r, w); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
	}
}

func GetFlashes(w http.ResponseWriter, r *http.Request) []interface{} {
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return nil
	}
	flashes := session.Flashes()
	if err := session.Save(r, w); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return nil
	}
	return flashes
}

func GetSession(r *http.Request) (*sessions.Session, error) {
	session, _ := store.Get(r, "minitwit-session")
	return session, nil
}
