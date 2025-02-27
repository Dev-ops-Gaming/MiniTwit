package handlers

import (
	"database/sql"
	"net/http"

	"minitwit/db"
	"minitwit/models"
	"minitwit/utils"

	"github.com/gorilla/mux"
)

func UserTimelineHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		// For some reason favicon.ico is being passed as a username, it also changes the username to lowercase? - ignore this for now, fix later
		if vars["username"] == "favicon.ico" {
			return
		}

		username := vars["username"]
		profileUser, err := models.GetUserByUsername(database, username)
		if err != nil {
			http.Error(w, "User does not exist", http.StatusBadRequest)
			return
		}

		messages, err := db.QueryUserTimeline(database, username)
		if err != nil {
			http.Error(w, "Failed to load user timeline", http.StatusInternalServerError)
			return
		}

		// Default data
		data := struct {
			Messages    []models.Message
			User        *models.User
			PageType    string
			ProfileUser models.User
			Followed    bool
			Flashes     []interface{}
		}{
			Messages:    messages,
			User:        nil,
			PageType:    "user",
			ProfileUser: *profileUser,
			Followed:    false,
			Flashes:     utils.GetFlashes(w, r),
		}

		session, _ := utils.GetSession(r)

		// User is logged in
		if session.Values["user_id"] != nil {
			userID := session.Values["user_id"].(int)
			username := session.Values["username"].(string)
			data.User = &models.User{Username: username, ID: userID}
			data.Followed, err = db.IsUserFollowing(database, userID, profileUser.ID)
			if err != nil {
				http.Error(w, "Failed to check if user is following", http.StatusInternalServerError)
				return
			}
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}
