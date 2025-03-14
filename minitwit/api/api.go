package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"minitwit/db"
	"minitwit/models"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/gorm"
)

const DATABASE = "../minitwit.db"

func notReqFromSimulator(w http.ResponseWriter, r *http.Request) bool {
	from_simulator := r.Header.Get("Authorization")
	if from_simulator != "Basic c2ltdWxhdG9yOnN1cGVyX3NhZmUh" {
		w.WriteHeader(403)
		response := map[string]any{"status": 403, "error_msg": "You are not authorized to access this resource!"}
		json.NewEncoder(w).Encode(response)
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
		http.Error(w, "Failed to read the latest ID. Try reloading the page and try again.", http.StatusInternalServerError)
	}

	//we need to convert to int, otherwise tests fail
	latest := string(content)
	latestInt, err := strconv.Atoi(latest)
	if err != nil {
		http.Error(w, "Failed to convert string to int.", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"latest": latestInt})
}

func register(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		//must decode into struct bc data sent as json, which golang bitches abt
		d := json.NewDecoder(r.Body)
		var t models.User
		d.Decode(&t)

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
				user := models.User{Username: t.Username, Email: t.Email, Pw_hash: pwHash}
				result := database.Create(&user)
				if result.Error != nil {
					log.Fatalf("Failed to insert in db: %v", err)
				}
			}
		}

		if erro != "" {
			jsonSstring, _ := json.Marshal(map[string]any{
				"status":    400,
				"error_msg": erro,
			})
			w.WriteHeader(400)
			w.Write(jsonSstring)
		} else {
			//.Write() sends header with status OK if .WriteHeader() has not yet been called
			//so we can just send empty message to signal status OK
			w.Write(json.RawMessage(""))
		}
	}
}

func messages(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		if notReqFromSimulator(w, r) {
			return
		}

		noMsgs, err := strconv.Atoi(r.URL.Query().Get("no"))
		if err != nil || noMsgs <= 0 {
			noMsgs = 100
		}

		if r.Method == "GET" {
			var users []models.User
			//Ordering when preloading:
			//https://github.com/go-gorm/gorm/issues/3004
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
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(filtered_msgs)
		}
	}
}

func messages_per_user(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		w.Header().Set("Content-Type", "application/json")
		if notReqFromSimulator(w, r) {
			return
		}

		//get the username
		vars := mux.Vars(r)
		username := vars["username"]
		noMsgs, err := strconv.Atoi(r.URL.Query().Get("no"))
		if err != nil || noMsgs <= 0 {
			noMsgs = 100
		}

		if r.Method == "GET" {
			user_id, err := db.GormGetUserId(database, username)
			if err != nil {
				print("user id not found in db!")
				panic(404)
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

			err = json.NewEncoder(w).Encode(filtered_msgs)
			if err != nil {
				fmt.Println("failed to convert messages to json")
				print(err.Error())
			}
		} else if r.Method == "POST" { // post message as <username>
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)
			content := req["content"]

			user_id, err := db.GormGetUserId(database, username)
			if err != nil {
				print("user id not found in db!")
				panic(404)
			}
			message := models.Message{Author_id: uint(user_id), Text: content.(string), Pub_date: time.Now().Unix()}

			result := database.Create(&message)
			if result.Error != nil {
				log.Fatalf("Failed to insert in db: %v", result.Error)
			}
			w.WriteHeader(204)
		}
	}
}

func follow(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		w.Header().Set("Content-Type", "application/json")
		if notReqFromSimulator(w, r) {
			return
		}

		//get the username
		vars := mux.Vars(r)
		username := vars["username"]
		user_id, err := db.GormGetUserId(database, username)
		if err != nil {
			print("user id not found in db!")
			panic(404)
		}

		noMsgs, err := strconv.Atoi(r.URL.Query().Get("no"))
		if err != nil || noMsgs <= 0 {
			noMsgs = 100
		}

		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)

		if r.Method == "POST" && req["follow"] != "" {
			follows_username := req["follow"]
			follows_user_id, err := db.GormGetUserId(database, follows_username)
			if err != nil {
				print("user id not found in db!")
				// TODO: This has to be another error, likely 500???
				panic(404)
			}

			follower := models.Follower{Who_id: user_id, Whom_id: follows_user_id}
			result := database.Create(&follower)
			if result.Error != nil {
				log.Fatalf("Failed to insert in db: %v", result.Error)
			}
			w.Write([]byte("204"))

		} else if r.Method == "POST" && req["unfollow"] != "" {
			unfollows_username := req["unfollow"]
			unfollows_user_id, err := db.GormGetUserId(database, unfollows_username)
			if err != nil {
				print("user id not found in db!")
				// TODO: This has to be another error, likely 500???
				panic(404)
			}

			err = database.Where("who_id=? AND whom_id=?", user_id, unfollows_user_id).Delete(&models.Follower{}).Error
			if err != nil {
				fmt.Printf("Failed to delete from db: %v", err)
			}
			w.Write([]byte("204"))

		} else if r.Method == "GET" {
			var users []models.User
			//get usernames of users whom given user is following
			database.Model(&models.User{}).Preload("Followers").Where("user_id=?", user_id).Limit(noMsgs).Find(&users)

			var follower_names []string
			for _, user := range users {
				for _, follows := range user.Followers {
					follower_names = append(follower_names, follows.Username)
				}
			}
			followers_response := map[string]any{"follows": follower_names}
			json.NewEncoder(w).Encode(followers_response)
		}
	}
}

func main() {
	// Db logic
	//this MUST be called, otherwise tests fail
	//seems grom cant read already existing database w/out migration stuff
	db.AutoMigrateDB()
	gorm_db := db.Gorm_ConnectDB()

	r := mux.NewRouter()

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
