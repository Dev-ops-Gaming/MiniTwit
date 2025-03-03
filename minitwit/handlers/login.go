package handlers

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"text/template"

	"minitwit/gorm_models"
	"minitwit/utils"

	"gorm.io/gorm"
)

func LoginHandler(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, _ := utils.GetSession(r)

		if r.Method == "GET" {
			if store.Values["user_id"] != nil {
				http.Redirect(w, r, "/", http.StatusFound)
			}
			loginTmpl := template.Must(template.ParseFiles("templates/layout.html", "templates/login.html"))
			if err := loginTmpl.Execute(w, nil); err != nil {
				http.Error(w, "Failed to render template", http.StatusInternalServerError)
			}
			return
		}

		if r.Method == "POST" {
			// Get input from form
			username := r.FormValue("username")
			password := r.FormValue("password")

			//check if user exists
			/*user, err := models.GetUserByUsername(database, username)
			if err != nil {
				http.Error(w, "Error getting user from db", http.StatusInternalServerError)
				fmt.Println("Error getting user from db")
				return
			}*/
			user, err := gorm_models.GetUserByUsername(database, username)
			if err != nil {
				http.Error(w, "Error getting user from db", http.StatusInternalServerError)
				fmt.Println("Error getting user from db")
				return
			}
			// compare the given password with the hashed password in the database
			hash := md5.New()
			hash.Write([]byte(password))
			pwHash := hex.EncodeToString(hash.Sum(nil))

			if pwHash != user.Pw_hash {
				http.Error(w, "Invalid password", http.StatusBadRequest)
				fmt.Println("Invalid password")
				return
			}

			// Set session values
			store.Values["user_id"] = user.User_id
			store.Values["username"] = user.Username
			err = store.Save(r, w)

			// Redirect to timeline
			utils.AddFlash(w, r, "You were logged in")
			http.Redirect(w, r, "/", http.StatusFound)
		}
	}
}
