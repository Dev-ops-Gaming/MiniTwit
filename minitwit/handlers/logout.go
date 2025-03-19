package handlers

import (
	"net/http"

	"minitwit/utils"
)

func LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := utils.GetSession(r)
		if session.Values["user_id"] == nil {
			http.Error(w, "You are not logged in", http.StatusBadRequest)
			return
		}
		// TODO: doesnt work atm - i suspect it's because the session is being cleared, but not sure
		utils.AddFlash(w, r, "You have been logged out")

		session.Options.MaxAge = -1 // Clear session
		if err := session.Save(r, w); err != nil {
			http.Error(w, "Failed to save session", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusFound)
	}
}
