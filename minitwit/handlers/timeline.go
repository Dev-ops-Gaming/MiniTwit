package handlers

import (
	"database/sql"
	"net/http"
	"text/template"

	"minitwit/db"
	"minitwit/models"
	"minitwit/utils"
)

var tmpl = template.Must(template.New("layout.html").Funcs(template.FuncMap{
	"getGravatar": utils.GetGravatar, // Register the getGravatar function with the template - ugly but can't find a better way
}).ParseFiles("templates/layout.html", "templates/timeline.html"))

func TimelineHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := utils.GetSession(r)
		if err != nil {
			http.Error(w, "Failed to get session", http.StatusInternalServerError)
			return
		}

		if session.Values["user_id"] == nil || session.Values["username"] == nil {
			http.Redirect(w, r, "/public", http.StatusFound)
			return
		}

		userID := session.Values["user_id"].(int)
		username := session.Values["username"].(string)

		messages, err := db.QueryTimeline(database, userID)

		if err != nil {
			http.Error(w, "Failed to load timeline", http.StatusInternalServerError)
			return
		}

		data := struct {
			Messages []models.Message
			User     models.User
			PageType string
			Flashes  []interface{}
		}{
			Messages: messages,
			User:     models.User{Username: username, ID: userID},
			PageType: "timeline",
			Flashes:  utils.GetFlashes(w, r),
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}
