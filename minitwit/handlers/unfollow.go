package handlers

import (
	"database/sql"
	"net/http"

	"minitwit/models"
	"minitwit/utils"

	"github.com/gorilla/mux"
)

func UnfollowHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := utils.GetSession(r, w)
		if session.Values["user_id"] == nil {
			utils.AddFlash(w, r, "You are not logged in")
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Get the user to unfollow
		vars := mux.Vars(r)
		username := vars["username"]
		user, err := models.GetUserByUsername(database, username)

		if err != nil {
			http.Error(w, "User does not exist", http.StatusBadRequest)
			return
		}

		// Delete the follow from the database
		_, err = database.Exec("DELETE FROM follower WHERE who_id = ? AND whom_id = ?", session.Values["user_id"], user.ID)
		if err != nil {
			http.Error(w, "Failed to unfollow user", http.StatusInternalServerError)
			return
		}

		// Redirect to the user's timeline
		utils.AddFlash(w, r, "You have unfollowed "+username)
		http.Redirect(w, r, "/"+username, http.StatusFound)
	}
}
