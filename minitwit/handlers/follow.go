package handlers

import (
	"database/sql"
	"net/http"

	"minitwit/db"
	"minitwit/models"
	"minitwit/utils"

	"github.com/gorilla/mux"
)

func FollowHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := utils.GetSession(r)
		if session.Values["user_id"] == nil {
			utils.AddFlash(w, r, "You must be logged in to follow users")
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Get the user to follow
		vars := mux.Vars(r)
		username := vars["username"]
		user, err := models.GetUserByUsername(database, username)
		if err != nil {
			http.Error(w, "User does not exist", http.StatusBadRequest)
			return
		}

		// Check if the user is already following the user
		isFollowing, err := db.IsUserFollowing(database, session.Values["user_id"].(int), user.ID)
		if err != nil {
			http.Error(w, "Failed to check if user is following", http.StatusInternalServerError)
			return
		}
		if isFollowing {
			utils.AddFlash(w, r, "You are already following "+username)
			http.Redirect(w, r, "/"+username, http.StatusFound)
			return
		}

		// Insert the follow into the database
		_, err = database.Exec("INSERT INTO follower (who_id, whom_id) VALUES (?, ?)", session.Values["user_id"], user.ID)
		if err != nil {
			http.Error(w, "Failed to follow user", http.StatusInternalServerError)
			return
		}

		// Redirect to the user's timeline
		utils.AddFlash(w, r, "You are now following "+username)
		http.Redirect(w, r, "/"+username, http.StatusFound)
	}
}
