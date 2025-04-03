package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"minitwit/db"
	"minitwit/middleware"
	"minitwit/models"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

func afterReq(database *gorm.DB) {
	//Closes the database again at the end of the request.
	rawDB, err := database.DB()
	if err != nil {
		fmt.Println("Error getting *sql.DB object:", err)
		return
	}
	rawDB.Close()
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	response := map[string]any{"status": code, "error_msg": message}
	middleware.RecordResponseMessage(code, message)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

func respondWithSuccess(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Failed to encode success response: %v", err)
	}
}

func notReqFromSimulator(w http.ResponseWriter, r *http.Request) bool {
	from_simulator := r.Header.Get("Authorization")
	if from_simulator != "Basic c2ltdWxhdG9yOnN1cGVyX3NhZmUh" {
		respondWithError(w, http.StatusForbidden, "You are not authorized to access this resource!")
		return true
	}
	return false
}

func updateLatest(r *http.Request) {
	// Get arg value associated with 'latest' & convert to int
	parsed_command_id := r.FormValue("latest")
	if parsed_command_id != "-1" && parsed_command_id != "" {
		//f, err := os.OpenFile("./latest_processed_sim_action_id.txt", os.O_WRONLY, os.)//, os.ModeAppend)
		f, err := os.Create("./latest_processed_sim_action_id.txt")
		if err != nil {
			log.Fatalf("Failed to read latest_id file: %v", err)
		}

		//this returns an int as well, but not sure what its used to signal lol
		_, err = f.WriteString(parsed_command_id)
		if err != nil {
			log.Fatalf("Failed to convert write id to file: %v", err)
		}
	}
}

func getLatest(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile("./latest_processed_sim_action_id.txt")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to read the latest ID. Try reloading the page and try again.")
		return
	}

	//we need to convert to int, otherwise tests fail
	latest := string(content)
	latestInt, err := strconv.Atoi(latest)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to convert string to int.")
		return
	}

	respondWithSuccess(w, http.StatusOK, map[string]int{"latest": latestInt})
}

func register(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		//must decode into struct bc data sent as json, which golang bitches abt
		d := json.NewDecoder(r.Body)
		var t models.User
		if err := d.Decode(&t); err != nil {
			respondWithError(w, http.StatusBadRequest, "Failed to decode request body.")
			return
		}

		var erro string = ""
		if r.Method == "POST" {
			if t.Username == "" {
				erro = "You have to enter a username"
			} else if t.Email == "" || !strings.ContainsAny(t.Email, "@") {
				erro = "You have to enter a valid email address"
			} else if t.Pwd == "" {
				erro = "You have to enter a password"
			} else if _, err := db.GormGetUserId(database, t.Username); err == nil {
				erro = "The username is already taken"
			} else {
				// hash the password
				hash := md5.New()
				hash.Write([]byte(r.Form.Get("pwd")))
				pwHash := hex.EncodeToString(hash.Sum(nil))
				// insert the user into the database
				user := models.User{Username: t.Username, Email: t.Email, PwHash: pwHash}
				result := database.Create(&user)
				if result.Error != nil {
					respondWithError(w, http.StatusInternalServerError, "Failed to insert in database.")
					return
				}
			}
		}

		if erro != "" {
			respondWithError(w, http.StatusBadRequest, erro)
		} else {
			w.WriteHeader(http.StatusCreated) // return 201
			afterReq(database)
		}
	}
}

func messages(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		if notReqFromSimulator(w, r) {
			return
		}

		// no_msgs = request.args.get("no", type=int, default=100)
		noMsgs, err := strconv.Atoi(r.URL.Query().Get("no"))
		if err != nil || noMsgs <= 0 {
			noMsgs = 100
		}

		if r.Method == "GET" {
			// modified the given API to remove some unnecessary select
			// might improve performance a bit
			var users []models.User
			database.Model(&models.User{}).Preload("Messages", "flagged = 0", func(database *gorm.DB) *gorm.DB {
				db := database.Order("pub_date DESC")
				return db
			}).Limit(noMsgs).Find(&users)

			var filtered_msgs []map[string]any
			for _, user := range users {
				for _, message := range user.Messages {
					filtered_msg := make(map[string]any)
					filtered_msg["content"] = message.Text
					filtered_msg["pub_date"] = message.Pub_date
					filtered_msg["user"] = user.Username
					filtered_msgs = append(filtered_msgs, filtered_msg)
				}
			}
			respondWithSuccess(w, http.StatusOK, filtered_msgs)
		}
		afterReq(database)
	}
}

func messages_per_user(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		if notReqFromSimulator(w, r) {
			return
		}

		vars := mux.Vars(r)
		username := vars["username"]
		noMsgs, err := strconv.Atoi(r.URL.Query().Get("no"))
		if err != nil || noMsgs <= 0 {
			noMsgs = 100
		}

		if r.Method == "GET" {
			user_id, err := db.GormGetUserId(database, username)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "User not found.")
				return
			}

			var users []models.User
			database.Model(&models.User{}).Preload("Messages", "flagged = 0", func(database *gorm.DB) *gorm.DB {
				db := database.Order("pub_date DESC")
				return db
			}).Where("user_id = ?", user_id).Limit(noMsgs).Find(&users)

			var filtered_msgs []map[string]any
			for _, user := range users {
				for _, message := range user.Messages {
					filtered_msg := make(map[string]any)
					filtered_msg["content"] = message.Text
					filtered_msg["pub_date"] = message.Pub_date
					filtered_msg["user"] = user.Username
					filtered_msgs = append(filtered_msgs, filtered_msg)
				}
			}
			respondWithSuccess(w, http.StatusOK, filtered_msgs)
		} else if r.Method == "POST" {
			var req map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				respondWithError(w, http.StatusBadRequest, "Failed to decode request body.")
				return
			}
			content := req["content"]

			user_id, err := db.GormGetUserId(database, username)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "User not found.")
				return
			}
			message := models.Message{Author_id: uint(user_id), Text: content.(string), Pub_date: time.Now().Unix()}

			result := database.Create(&message)
			if result.Error != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to insert in database.")
				return
			}
			w.WriteHeader(204)
		}
		afterReq(database)
	}
}

func follow(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		if notReqFromSimulator(w, r) {
			return
		}

		vars := mux.Vars(r)
		username := vars["username"]
		user_id, err := db.GormGetUserId(database, username)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "User not found.")
			return
		}

		noMsgs, err := strconv.Atoi(r.URL.Query().Get("no"))
		if err != nil || noMsgs <= 0 {
			noMsgs = 100
		}

		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Failed to decode request body.")
			return
		}

		if r.Method == "POST" && req["follow"] != "" {
			follows_username := req["follow"]
			follows_user_id, err := db.GormGetUserId(database, follows_username)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "User not found.")
				return
			}

			follower := models.Follower{Who_id: user_id, Whom_id: follows_user_id}
			result := database.Create(&follower)
			if result.Error != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to insert in database.")
				return
			}
			w.WriteHeader(http.StatusNoContent)
		} else if r.Method == "POST" && req["unfollow"] != "" {
			unfollows_username := req["unfollow"]
			unfollows_user_id, err := db.GormGetUserId(database, unfollows_username)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "User not found.")
				return
			}

			err = database.Where("who_id=? AND whom_id=?", user_id, unfollows_user_id).Delete(&models.Follower{}).Error
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to delete from database.")
				return
			}
			w.WriteHeader(http.StatusNoContent)

		} else if r.Method == "GET" {
			var users []models.User
			database.Model(&models.User{}).Preload("Followers").Where("user_id=?", user_id).Limit(noMsgs).Find(&users)

			var follower_names []string
			for _, user := range users {
				for _, follows := range user.Followers {
					follower_names = append(follower_names, follows.Username)
				}
			}
			followers_response := map[string]any{"follows": follower_names}
			respondWithSuccess(w, http.StatusOK, followers_response)
		}
		afterReq(database)
	}
}

func main() {
	// Db logic
	//this MUST be called, otherwise tests fail
	//seems grom cant read already existing database w/out migration stuff
	db.AutoMigrateDB()
	gorm_db := db.Gorm_ConnectDB()

	r := mux.NewRouter()

	// Middleware
	r.Use(middleware.PrometheusMiddleware)

	// expose metrics
	r.Handle("/metrics", promhttp.Handler())

	// Define routes
	r.HandleFunc("/register", register(gorm_db)).Methods("POST")
	r.HandleFunc("/latest", getLatest).Methods("GET")
	r.HandleFunc("/msgs", messages(gorm_db)).Methods("GET")
	r.HandleFunc("/msgs/{username}", messages_per_user(gorm_db)).Methods("GET", "POST")
	r.HandleFunc("/fllws/{username}", follow(gorm_db)).Methods("GET", "POST")

	// Start the server
	fmt.Println("API is running on http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", r))
}
