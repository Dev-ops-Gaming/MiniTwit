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

var noUserFoundError = "User not found."
var DecodeError = "Failed to decode request body."
var dbInsertError = "Failed to insert in database."

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
	fromSimulator := r.Header.Get("Authorization")
	if fromSimulator != "Basic c2ltdWxhdG9yOnN1cGVyX3NhZmUh" {
		respondWithError(w, http.StatusForbidden, "You are not authorized to access this resource!")
		return true
	}
	return false
}

func updateLatest(r *http.Request) {
	// Get arg value associated with 'latest' & convert to int
	parsedCommandId := r.FormValue("latest")
	if parsedCommandId != "-1" && parsedCommandId != "" {
		//f, err := os.OpenFile("./latest_processed_sim_action_id.txt", os.O_WRONLY, os.)//, os.ModeAppend)
		f, err := os.Create("./latest_processed_sim_action_id.txt")
		if err != nil {
			log.Fatalf("Failed to read latest_id file: %v", err)
		}

		//this returns an int as well, but not sure what its used to signal lol
		_, err = f.WriteString(parsedCommandId)
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

func checkRegisterUserInput(t models.User, database *gorm.DB) string {
	var erro string = ""
	if t.Username == "" {
		erro = "You have to enter a username"
	} else if t.Email == "" || !strings.ContainsAny(t.Email, "@") {
		erro = "You have to enter a valid email address"
	} else if t.Pwd == "" {
		erro = "You have to enter a password"
	} else if _, err := db.GormGetUserId(database, t.Username); err == nil {
		erro = "The username is already taken"
	}

	return erro
}

func register(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		//must decode into struct bc data sent as json, which golang bitches abt
		d := json.NewDecoder(r.Body)
		var t models.User
		if err := d.Decode(&t); err != nil {
			respondWithError(w, http.StatusBadRequest, DecodeError)
			return
		}

		var erro string = ""
		if r.Method == "POST" {
			erro = checkRegisterUserInput(t, database)
			//If input ok, register user in db
			if erro == "" {
				// hash the password
				hash := md5.New()
				hash.Write([]byte(r.Form.Get("pwd")))
				pwHash := hex.EncodeToString(hash.Sum(nil))
				// insert the user into the database
				user := models.User{Username: t.Username, Email: t.Email, PwHash: pwHash}
				result := database.Create(&user)
				if result.Error != nil {
					respondWithError(w, http.StatusInternalServerError, dbInsertError)
					return
				}
			}
		}

		if erro != "" {
			respondWithError(w, http.StatusBadRequest, erro)
		} else {
			w.WriteHeader(http.StatusCreated) // return 201
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

			var filteredMsgs []map[string]any
			for _, user := range users {
				for _, message := range user.Messages {
					filteredMsg := make(map[string]any)
					filteredMsg["content"] = message.Text
					filteredMsg["pub_date"] = message.Pub_date
					filteredMsg["user"] = user.Username
					filteredMsgs = append(filteredMsgs, filteredMsg)
				}
			}
			respondWithSuccess(w, http.StatusOK, filteredMsgs)
		}
	}
}

func messagesPerUserGET(w http.ResponseWriter, database *gorm.DB, username string, noMsgs int) {
	userId, err := db.GormGetUserId(database, username)
	if err != nil {
		respondWithError(w, http.StatusNotFound, noUserFoundError)
		return
	}

	var users []models.User
	database.Model(&models.User{}).Preload("Messages", "flagged = 0", func(database *gorm.DB) *gorm.DB {
		db := database.Order("pub_date DESC")
		return db
	}).Where("user_id = ?", userId).Limit(noMsgs).Find(&users)

	var filteredMsgs []map[string]any
	for _, user := range users {
		for _, message := range user.Messages {
			filteredMsg := make(map[string]any)
			filteredMsg["content"] = message.Text
			filteredMsg["pub_date"] = message.Pub_date
			filteredMsg["user"] = user.Username
			filteredMsgs = append(filteredMsgs, filteredMsg)
		}
	}
	respondWithSuccess(w, http.StatusOK, filteredMsgs)
}

func messagesPerUserPOST(w http.ResponseWriter, r *http.Request, database *gorm.DB, username string) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, DecodeError)
		return
	}
	content := req["content"]

	userId, err := db.GormGetUserId(database, username)
	if err != nil {
		respondWithError(w, http.StatusNotFound, noUserFoundError)
		return
	}
	message := models.Message{Author_id: uint(userId), Text: content.(string), Pub_date: time.Now().Unix()}

	result := database.Create(&message)
	if result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, dbInsertError)
		return
	}
	w.WriteHeader(204)
}

func messagesPerUser(database *gorm.DB) http.HandlerFunc {
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
			messagesPerUserGET(w, database, username, noMsgs)

		} else if r.Method == "POST" {
			messagesPerUserPOST(w, r, database, username)
		}
	}
}

func followUser(database *gorm.DB, w http.ResponseWriter, curUserId int, toFollowUsername string) {
	followsUsername := toFollowUsername
	followsUserId, err := db.GormGetUserId(database, followsUsername)
	if err != nil {
		respondWithError(w, http.StatusNotFound, noUserFoundError)
		return
	}

	follower := models.Follower{Who_id: curUserId, Whom_id: followsUserId}
	result := database.Create(&follower)
	if result.Error != nil {
		respondWithError(w, http.StatusInternalServerError, dbInsertError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func unfollowUser(database *gorm.DB, w http.ResponseWriter, curUserId int, toUnfollowUsername string) {
	unfollowsUsername := toUnfollowUsername
	unfollowsUserId, err := db.GormGetUserId(database, unfollowsUsername)
	if err != nil {
		respondWithError(w, http.StatusNotFound, noUserFoundError)
		return
	}

	err = database.Where("who_id=? AND whom_id=?", curUserId, unfollowsUserId).Delete(&models.Follower{}).Error
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete from database.")
		return
	}
	w.WriteHeader(http.StatusNoContent)

}

func getFollowers(database *gorm.DB, w http.ResponseWriter, curUserId int, noMsgs int) {
	var users []models.User
	database.Model(&models.User{}).Preload("Followers").Where("user_id=?", curUserId).Limit(noMsgs).Find(&users)

	var followerNames []string
	for _, user := range users {
		for _, follows := range user.Followers {
			followerNames = append(followerNames, follows.Username)
		}
	}
	followersResponse := map[string]any{"follows": followerNames}
	respondWithSuccess(w, http.StatusOK, followersResponse)
}

func follow(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		if notReqFromSimulator(w, r) {
			return
		}

		vars := mux.Vars(r)
		username := vars["username"]
		userId, err := db.GormGetUserId(database, username)
		if err != nil {
			respondWithError(w, http.StatusNotFound, noUserFoundError)
			return
		}

		noMsgs, err := strconv.Atoi(r.URL.Query().Get("no"))
		if err != nil || noMsgs <= 0 {
			noMsgs = 100
		}

		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, DecodeError)
			return
		}

		if r.Method == "POST" && req["follow"] != "" {
			followsUsername := req["follow"]
			followUser(database, w, userId, followsUsername)

		} else if r.Method == "POST" && req["unfollow"] != "" {
			unfollowsUsername := req["unfollow"]
			unfollowUser(database, w, userId, unfollowsUsername)

		} else if r.Method == "GET" {
			getFollowers(database, w, userId, noMsgs)
		}
	}
}

func main() {
	// Db logic
	//this MUST be called, otherwise tests fail
	//seems grom cant read already existing database w/out migration stuff
	db.AutoMigrateDB()
	gormDB := db.GormConnectDB()

	r := mux.NewRouter()

	// Middleware
	r.Use(middleware.PrometheusMiddleware)

	// expose metrics
	r.Handle("/metrics", promhttp.Handler())

	// Define routes
	r.HandleFunc("/register", register(gormDB)).Methods("POST")
	r.HandleFunc("/latest", getLatest).Methods("GET")
	r.HandleFunc("/msgs", messages(gormDB)).Methods("GET")
	r.HandleFunc("/msgs/{username}", messagesPerUser(gormDB)).Methods("GET", "POST")
	r.HandleFunc("/fllws/{username}", follow(gormDB)).Methods("GET", "POST")

	// Start the server
	fmt.Println("API is running on http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", r))
}
