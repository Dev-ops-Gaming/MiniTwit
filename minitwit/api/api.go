package main

import (
	"crypto/md5"
	"database/sql"
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
)

// PLS DONT DELETE THIS LINE 3:
// pytest -k "test_latest" -s
var database *sql.DB

const DBPATH = "../minitwit.db"

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

func getUserId(database *sql.DB, username string) (int, any) {
	// Convenience method to look up the id for a username.
	var userId int
	if err := database.QueryRow("select user_id from user where username = ?", username).Scan(&userId); err == sql.ErrNoRows {
		//no rows
		return -1, err
	} else {
		//got something
		return userId, nil
	}
}

func updateLatest(r *http.Request) {
	// Get arg value associated with 'latest' & convert to int
	//parsed_command_id, err := strconv.Atoi(r.FormValue("latest"))
	parsed_command_id := r.FormValue("latest")
	//var f *os.File
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

// verified working
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

// should be refactored
func register(database *sql.DB) http.HandlerFunc { //([]byte, int)
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		//must decode into struct bc data sent as json, which golang bitches abt
		d := json.NewDecoder(r.Body)
		var t models.User
		d.Decode(&t)
		//fmt.Println(t)
		var erro string = ""
		if r.Method == "POST" {
			if t.Username == "" {
				erro = "You have to enter a username"
			} else if t.Email == "" || !strings.ContainsAny(t.Email, "@") {
				erro = "You have to enter a valid email address"
			} else if t.Pwd == "" {
				erro = "You have to enter a password"
				//else if get_user_id not none is missing
			} else if id, err := getUserId(database, t.Username); err == nil || id != -1 {
				erro = "The username is already taken"
				fmt.Println(id)
			} else {
				// hash the password
				hash := md5.New()
				hash.Write([]byte(r.Form.Get("pwd")))
				pwHash := hex.EncodeToString(hash.Sum(nil))
				// insert the user into the database
				/*_, err := database.Exec("INSERT INTO user (username, email, pw_hash) VALUES (?, ?, ?)", r.Form.Get("username"), r.Form.Get("email"), pwHash)*/
				_, err := database.Exec("INSERT INTO user (username, email, pw_hash) VALUES (?, ?, ?)", t.Username, t.Email, pwHash)
				if err != nil {
					log.Fatalf("Failed to insert in db: %v", err)
				}
			}
		}

		if erro != "" {
			jsonSstring, _ := json.Marshal(map[string]any{
				"status":    400,
				"error_msg": erro,
			})
			fmt.Println(erro)
			w.WriteHeader(400)
			w.Write(jsonSstring)
			//return jsonSstring, 400
		} else {
			//[]byte("", "204")
			//w.Write(json.RawMessage(""), 204)
			//return json.RawMessage(""), 204
			w.Write(json.RawMessage(""))
		}
	}
}

// verified working
func messages(database *sql.DB) http.HandlerFunc {
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
			rows, err := db.QueryDB(database,
				`SELECT message.text, message.pub_date, user.username
				 FROM message JOIN user ON message.author_id = user.user_id
				 WHERE message.flagged = 0
				 ORDER BY message.pub_date DESC
				 LIMIT ?`, noMsgs)
			if err != nil {
				http.Error(w, `{"status":500, "error_msg":"Database query failed"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var filteredMsgs []map[string]any
			for rows.Next() {
				var text, pubDate, username string
				if err := rows.Scan(&text, &pubDate, &username); err != nil {
					http.Error(w, `{"status":500, "error_msg":"Failed to scan database results"}`, http.StatusInternalServerError)
					return
				}

				filteredMsgs = append(filteredMsgs, map[string]any{
					"content":  text,
					"pub_date": pubDate,
					"user":     username,
				})
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(filteredMsgs)
		}
	}
}

// verified working
func messages_per_user(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		if notReqFromSimulator(w, r) {
			return
		}

		// get the username from the URL
		vars := mux.Vars(r)
		username := vars["username"]

		// get the user id from the database
		userID, err := getUserId(database, username)
		if err != nil {
			http.Error(w, `{"status":404, "error_msg":"User not found"}`, 404)
			return
		}

		if r.Method == "GET" {

			// Parse the 'no' query parameter, default to 100
			noMsgs, err := strconv.Atoi(r.URL.Query().Get("no"))
			if err != nil || noMsgs <= 0 {
				noMsgs = 100
			}

			query := `SELECT message.text, message.pub_date, user.username
                      FROM message JOIN user ON message.author_id = user.user_id
                      WHERE message.flagged = 0 AND user.user_id = ?
                      ORDER BY message.pub_date DESC LIMIT ?`

			rows, err := database.Query(query, userID, noMsgs)
			if err != nil {
				http.Error(w, `{"status":500, "error_msg":"Database query failed"}`, 500)
				log.Println("Database query failed:", err)
				return
			}
			defer rows.Close()

			// Process query results
			var filteredMsgs []map[string]any
			for rows.Next() {
				var text, pubDate, username string
				if err := rows.Scan(&text, &pubDate, &username); err != nil {
					http.Error(w, `{"status":500, "error_msg":"Failed to process database results"}`, 500)
					log.Println("Failed to scan database results:", err)
					return
				}

				filteredMsgs = append(filteredMsgs, map[string]any{
					"content":  text,
					"pub_date": pubDate,
					"user":     username,
				})
			}

			// Send response
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(filteredMsgs)

		} else if r.Method == "POST" { // post message as <username>
			// another struct that maybe should be in models
			var req struct {
				Content string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"status":400, "error_msg":"Invalid request format"}`, 400)
				return
			}

			query := `INSERT INTO message (author_id, text, pub_date, flagged) VALUES (?, ?, ?, 0)`
			_, err := database.Exec(query, userID, req.Content, time.Now().Unix())
			if err != nil {
				http.Error(w, `{"status":500, "error_msg":"Failed to insert message"}`, 500)
				log.Println("Failed to insert message:", err)
				return
			}
			w.WriteHeader(204) // success - return empty response with status 204
		}
	}
}

func follow(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		w.Header().Set("Content-Type", "application/json")
		if notReqFromSimulator(w, r) {
			return
		}

		//get the username
		vars := mux.Vars(r)
		username := vars["username"]
		user_id, _ := getUserId(database, username)
		if user_id == -1 {
			fmt.Println("follow")
			fmt.Println(username)
			print("user id not found in db!")
			//abort(404)
		}
		//no_followers := r.FormValue("no")

		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		//content := req["content"]

		if r.Method == "POST" && req["follow"] != "" {
			fmt.Println("POST and follow!")
			follows_username := req["follow"] //r.FormValue("follow")
			follows_user_id, _ := getUserId(database, follows_username)
			if follows_user_id == -1 {
				// TODO: This has to be another error, likely 500???
				//abort(404)
				print("lol error 404 hehe")
			}
			query := "INSERT INTO follower (who_id, whom_id) VALUES (?, ?)"
			_, err := database.Exec(query, user_id, follows_user_id)
			if err != nil {
				log.Fatalf("Failed to insert in db: %v", err)
			}
			//return "", 204
			w.Write([]byte("204"))
		} else if r.Method == "POST" && req["unfollow"] != "" {
			fmt.Println("POST and UNfollow!")
			unfollows_username := req["unfollow"] //r.FormValue("unfollow")
			unfollows_user_id, _ := getUserId(database, unfollows_username)
			if unfollows_user_id == -1 {
				// TODO: This has to be another error, likely 500???
				//abort(404)
				print("lol error 404 hehe")
			}
			query := "DELETE FROM follower WHERE who_id=? and WHOM_ID=?"
			database.Exec(query, user_id, unfollows_user_id)
			//return "", 204
			w.Write([]byte("204"))
		} else if r.Method == "GET" {
			no_followers := r.FormValue("no")
			query := "SELECT user.username FROM user INNER JOIN follower ON follower.whom_id=user.user_id WHERE follower.who_id=? LIMIT ?"
			followers, _ := database.Query(query, user_id, no_followers)

			var follower_names []string
			for followers.Next() {
				var username string
				err := followers.Scan(&username)
				if err != nil {
					print(err.Error())
				}
				follower_names = append(follower_names, username)
			}
			followers_response := map[string]any{"follows": follower_names}
			fmt.Println(followers_response)
			//w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(followers_response)
		}
	}
}

func main() {
	// Db logic
	database, err := db.ConnectDB(DBPATH)
	if err != nil && database == nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer database.Close()

	r := mux.NewRouter()

	// Define routes
	r.HandleFunc("/register", register(database)).Methods("POST")
	r.HandleFunc("/latest", getLatest).Methods("GET") //no db
	r.HandleFunc("/msgs", messages(database)).Methods("GET")
	r.HandleFunc("/msgs/{username}", messages_per_user(database)).Methods("GET", "POST")
	r.HandleFunc("/fllws/{username}", follow(database)).Methods("GET", "POST")

	// Start the server
	fmt.Println("API is running on http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", r))
}
