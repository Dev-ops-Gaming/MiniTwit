package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/Dev-ops-Gaming/MiniTwit/utils"
)

func AddMessageHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, _ := utils.GetSession(r)
		if store.Values["user_id"] == nil {
			http.Error(w, "You are not logged in", http.StatusBadRequest)
			return
		}

		// Get input from form
		text := r.FormValue("text")
		userID := store.Values["user_id"].(int)

		// Insert message into the database
		_, err := database.Exec("INSERT INTO message (author_id, text, pub_date, flagged) VALUES (?, ?, ?, 0)", userID, text, time.Now().Unix())
		if err != nil {
			http.Error(w, "Failed to insert message", http.StatusInternalServerError)
			return
		}

		// Redirect to timeline
		utils.AddFlash(w, r, "Your message was recorded")
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
