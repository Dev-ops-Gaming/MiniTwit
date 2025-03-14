package handlers

import (
	"net/http"

	"minitwit/db"
	"minitwit/models"
	"minitwit/utils"

	"gorm.io/gorm"
)

func PublicTimelineHandler(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		messages, err := db.QueryPublicTimeline(database)
		if err != nil {
			http.Error(w, "Failed to load public timeline", http.StatusInternalServerError)
			return
		}

		// Default data
		data := struct {
			Messages []models.Message
			User     *models.User
			PageType string
			Flashes  []interface{}
		}{
			Messages: messages,
			User:     nil,
			PageType: "public",
			Flashes:  utils.GetFlashes(w, r),
		}

		session, _ := utils.GetSession(r)

		// User is logged in
		if session.Values["user_id"] != nil {
			userID := session.Values["user_id"].(int)
			username := session.Values["username"].(string)
			data.User = &models.User{Username: username, User_id: userID}
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}
