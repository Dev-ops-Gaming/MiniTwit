package handlers

import (
	"net/http"
	"time"

	"minitwit/models"
	"minitwit/utils"

	"gorm.io/gorm"
)

func AddMessageHandler(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, _ := utils.GetSession(r, w)
		if store.Values["user_id"] == nil {
			http.Error(w, "You are not logged in", http.StatusBadRequest)
			return
		}

		// Get input from form
		text := r.FormValue("text")
		userID := store.Values["user_id"].(int)

		// Insert message into the database
		message := models.Message{Author_id: uint(userID), Text: text, Pub_date: time.Now().Unix(), Flagged: 0}
		result := database.Create(&message)
		if result.Error != nil {
			http.Error(w, "Failed to insert message", http.StatusInternalServerError)
			return
		}

		// Redirect to timeline
		utils.AddFlash(w, r, "Your message was recorded")
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
