package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
)

const (
	DATABASE   = "./minitwit.db"
	PER_PAGE   = 32
	SECRET_KEY = "development key"
)

var (
	db   *sql.DB
	tmpl = template.Must(template.New("layout.html").Funcs(template.FuncMap{
		"getGravatar": getGravatar, // Register the getGravatar function with the template - ugly but can't find a better way
	}).ParseFiles("templates/layout.html", "templates/timeline.html"))
	registerTmpl = template.Must(template.ParseFiles("templates/layout.html", "templates/register.html"))
	store        = sessions.NewCookieStore([]byte(SECRET_KEY))
)

type User struct {
	ID       int
	Username string
	Email    string
	PwHash   string
}

type Message struct {
	ID      int
	Author  string
	Email   string
	Content string
	PubDate string
}

// Runner

func main() {
	// Db logic
	db = connectDB()
	defer db.Close()

	// Routes
	r := mux.NewRouter()
	r.HandleFunc("/", timelineHandler).Methods("GET")
	r.HandleFunc("/public", publicTimelineHandler).Methods("GET")
	r.HandleFunc("/register", registerHandler).Methods("GET", "POST")
	r.HandleFunc("/login", loginHandler).Methods("GET", "POST")
	r.HandleFunc("/logout", logoutHandler).Methods("GET")
	r.HandleFunc("/{username}", userTimelineHandler).Methods("GET")
	r.HandleFunc("/{username}/follow", followHandler).Methods("GET", "POST")
	r.HandleFunc("/{username}/unfollow", unfollowHandler).Methods("GET", "POST")
	r.HandleFunc("/add_message", addMessageHandler).Methods("POST")

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// General functions

func init_db() {
	db := connectDB()
	defer db.Close()

	// Read the schema file
	schemaPath := filepath.Join(filepath.Dir(os.Args[0]), "schema.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}

	// Execute the schema
	_, err = db.Exec(string(schema))
	if err != nil {
		log.Fatalf("Failed to execute schema: %v", err)
	}

}

func connectDB() *sql.DB {
	db, err := sql.Open("sqlite3", DATABASE)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	return db
}

func queryDB(query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func formatTime(timestamp int64) string {
	return time.Unix(timestamp, 0).Format("02-01-2006 15:04:05")
}

func getGravatar(email string, size int) string {
	// Return the gravatar image for the given email address.
	hash := md5.New()
	hash.Write([]byte(strings.ToLower(email)))
	return fmt.Sprintf("http://www.gravatar.com/avatar/%s?d=identicon&s=%d", hex.EncodeToString(hash.Sum(nil)), size)
}

// Flash messages
func addFlash(w http.ResponseWriter, r *http.Request, message string) {
	session, _ := store.Get(r, "minitwit-session")
	session.AddFlash(message)
	session.Save(r, w)
}

func getFlashes(w http.ResponseWriter, r *http.Request) []interface{} {
	session, _ := store.Get(r, "minitwit-session")
	flashes := session.Flashes()
	session.Save(r, w)
	return flashes
}

// Handlers

func timelineHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "minitwit-session")

	if session.Values["user_id"] == nil || session.Values["username"] == nil {
		http.Redirect(w, r, "/public", http.StatusFound)
		return
	}

	userID := session.Values["user_id"].(int)
	username := session.Values["username"].(string)

	messages, err := queryTimeline(userID)

	if err != nil {
		http.Error(w, "Failed to load timeline", http.StatusInternalServerError)
		return
	}

	data := struct {
		Messages []Message
		User     User
		PageType string
		Flashes  []interface{}
	}{
		Messages: messages,
		User:     User{Username: username, ID: userID},
		PageType: "timeline",
		Flashes:  getFlashes(w, r),
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func publicTimelineHandler(w http.ResponseWriter, r *http.Request) {
	messages, err := queryPublicTimeline()
	if err != nil {
		http.Error(w, "Failed to load public timeline", http.StatusInternalServerError)
		return
	}

	// Default data
	data := struct {
		Messages []Message
		User     *User
		PageType string
		Flashes  []interface{}
	}{
		Messages: messages,
		User:     nil,
		PageType: "public",
		Flashes:  getFlashes(w, r),
	}

	session, _ := store.Get(r, "minitwit-session")

	// User is logged in
	if session.Values["user_id"] != nil {
		userID := session.Values["user_id"].(int)
		username := session.Values["username"].(string)
		data.User = &User{Username: username, ID: userID}
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func userTimelineHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// For some reason favicon.ico is being passed as a username, it also changes the username to lowercase? - ignore this for now, fix later
	if vars["username"] == "favicon.ico" {
		return
	}

	username := vars["username"]
	profileUser, err := getUserFromDb(username)
	if err != nil {
		http.Error(w, "User does not exist", http.StatusBadRequest)
		return
	}

	messages, err := queryUserTimeline(username)
	if err != nil {
		http.Error(w, "Failed to load user timeline", http.StatusInternalServerError)
		return
	}

	// Default data
	data := struct {
		Messages    []Message
		User        *User
		PageType    string
		ProfileUser User
		Followed    bool
		Flashes     []interface{}
	}{
		Messages:    messages,
		User:        nil,
		PageType:    "user",
		ProfileUser: profileUser,
		Followed:    false,
		Flashes:     getFlashes(w, r),
	}

	session, _ := store.Get(r, "minitwit-session")

	// User is logged in
	if session.Values["user_id"] != nil {
		userID := session.Values["user_id"].(int)
		username := session.Values["username"].(string)
		data.User = &User{Username: username, ID: userID}
		data.Followed, err = isUserFollowing(userID, profileUser.ID)
		if err != nil {
			http.Error(w, "Failed to check if user is following", http.StatusInternalServerError)
			return
		}
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if err := registerTmpl.Execute(w, nil); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
	if r.Method == "POST" {
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")
		password2 := r.FormValue("password2")

		// input validation
		if username == "" || email == "" || password == "" {
			http.Error(w, "You must fill out all fields", http.StatusBadRequest)
		}

		// Check if repeated password matches
		if password != password2 {
			http.Error(w, "Passwords do not match", http.StatusBadRequest)
		}

		//check if user already exists
		_, err := getUserFromDb(username)
		if err == nil {
			http.Error(w, "User already exists", http.StatusBadRequest)
		}

		// hash the password
		hash := md5.New()
		hash.Write([]byte(password))
		pwHash := hex.EncodeToString(hash.Sum(nil))

		// insert the user into the database
		_, err = db.Exec("INSERT INTO user (username, email, pw_hash) VALUES (?, ?, ?)", username, email, pwHash)

		// redirect to timeline
		addFlash(w, r, "You were successfully registered and can login now")
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func addMessageHandler(w http.ResponseWriter, r *http.Request) {
	store, _ := store.Get(r, "minitwit-session")
	if store.Values["user_id"] == nil {
		http.Error(w, "You are not logged in", http.StatusBadRequest)
		return
	}

	// Get input from form
	text := r.FormValue("text")
	userID := store.Values["user_id"].(int)

	// Insert message into the database
	_, err := db.Exec("INSERT INTO message (author_id, text, pub_date, flagged) VALUES (?, ?, ?, 0)", userID, text, time.Now().Unix())
	if err != nil {
		http.Error(w, "Failed to insert message", http.StatusInternalServerError)
		return
	}

	// Redirect to timeline
	addFlash(w, r, "Your message was recorded")
	http.Redirect(w, r, "/", http.StatusFound)
}

//Query functions

func queryTimeline(userID int) ([]Message, error) {
	rows, err := queryDB(`
		select message.*, user.* 
		from message, user
        where message.flagged = 0 and message.author_id = user.user_id and (
            user.user_id = ? or
            user.user_id in (select whom_id from follower
                                    where who_id = ?))
		order by message.pub_date desc limit ?`, userID, userID, PER_PAGE)
	if err != nil {
		fmt.Println("Error in queryTimeline: ", err)
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		// Refactor later
		var pubDate int64
		var messageID, authorID, flagged, userID int
		var text, username, email, pwHash string

		err := rows.Scan(&messageID, &authorID, &text, &pubDate, &flagged, &userID, &username, &email, &pwHash)
		// Construct Message struct
		var m Message
		m = Message{ID: messageID, Author: username, Content: text, Email: email}
		m.PubDate = formatTime(pubDate) // Convert timestamp from UNIX to readable format
		if err != nil {
			fmt.Println("Error in queryTimeline: ", err)
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func queryPublicTimeline() ([]Message, error) {
	rows, err := queryDB(`
		SELECT message.author_id, user.username, message.text, message.pub_date, user.email
		FROM message
		JOIN user ON message.author_id = user.user_id
		WHERE message.flagged = 0
		ORDER BY message.pub_date DESC
		LIMIT ?`, PER_PAGE)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		var pubDate int64
		err := rows.Scan(&m.ID, &m.Author, &m.Content, &pubDate, &m.Email)
		m.PubDate = formatTime(pubDate) // Convert timestamp from UNIX to readable format
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func queryUserTimeline(username string) ([]Message, error) {
	rows, err := queryDB(`
		SELECT message.author_id, user.username, message.text, message.pub_date, user.email
		FROM message
		JOIN user ON message.author_id = user.user_id
		WHERE user.username = ? AND message.flagged = 0
		ORDER BY message.pub_date DESC
		LIMIT ?`, username, PER_PAGE)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		var pubDate int64
		err := rows.Scan(&m.ID, &m.Author, &m.Content, &pubDate, &m.Email)
		m.PubDate = formatTime(pubDate)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func getUserFromDb(username string) (User, error) {
	var u User
	row := db.QueryRow("SELECT * FROM user WHERE username = ?", username)
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PwHash)
	if err != nil {
		return u, err
	}
	return u, nil
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	store, _ := store.Get(r, "minitwit-session")

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
		user, err := getUserFromDb(username)
		if err != nil {
			http.Error(w, "Error getting user from db", http.StatusInternalServerError)
			fmt.Println("Error getting user from db")
			return
		}

		// compare the given password with the hashed password in the database
		hash := md5.New()
		hash.Write([]byte(password))
		pwHash := hex.EncodeToString(hash.Sum(nil))

		if pwHash != user.PwHash {
			http.Error(w, "Invalid password", http.StatusBadRequest)
			fmt.Println("Invalid password")
			return
		}

		// Set session values
		store.Values["user_id"] = user.ID
		store.Values["username"] = user.Username
		err = store.Save(r, w)

		// Redirect to timeline
		addFlash(w, r, "You were logged in")
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "minitwit-session")
	if session.Values["user_id"] == nil {
		http.Error(w, "You are not logged in", http.StatusBadRequest)
		return
	}
	// TODO: doesnt work atm - i suspect it's because the session is being cleared, but not sure
	addFlash(w, r, "You have been logged out")

	session.Options.MaxAge = -1 // Clear session
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

func followHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "minitwit-session")
	if session.Values["user_id"] == nil {
		addFlash(w, r, "You must be logged in to follow users")
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Get the user to follow
	vars := mux.Vars(r)
	username := vars["username"]
	user, err := getUserFromDb(username)
	if err != nil {
		http.Error(w, "User does not exist", http.StatusBadRequest)
		return
	}

	// Check if the user is already following the user
	isFollowing, err := isUserFollowing(session.Values["user_id"].(int), user.ID)
	if err != nil {
		http.Error(w, "Failed to check if user is following", http.StatusInternalServerError)
		return
	}
	if isFollowing {
		addFlash(w, r, "You are already following "+username)
		http.Redirect(w, r, "/"+username, http.StatusFound)
		return
	}

	// Insert the follow into the database
	_, err = db.Exec("INSERT INTO follower (who_id, whom_id) VALUES (?, ?)", session.Values["user_id"], user.ID)
	if err != nil {
		http.Error(w, "Failed to follow user", http.StatusInternalServerError)
		return
	}

	// Redirect to the user's timeline
	addFlash(w, r, "You are now following "+username)
	http.Redirect(w, r, "/"+username, http.StatusFound)
}

func unfollowHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "minitwit-session")
	if session.Values["user_id"] == nil {
		addFlash(w, r, "You are not logged in")
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Get the user to unfollow
	vars := mux.Vars(r)
	username := vars["username"]
	user, err := getUserFromDb(username)

	if err != nil {
		http.Error(w, "User does not exist", http.StatusBadRequest)
		return
	}

	// Delete the follow from the database
	_, err = db.Exec("DELETE FROM follower WHERE who_id = ? AND whom_id = ?", session.Values["user_id"], user.ID)
	if err != nil {
		http.Error(w, "Failed to unfollow user", http.StatusInternalServerError)
		return
	}

	// Redirect to the user's timeline
	addFlash(w, r, "You have unfollowed "+username)
	http.Redirect(w, r, "/"+username, http.StatusFound)
}

func isUserFollowing(whoID, whomID int) (bool, error) {
	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM follower WHERE who_id = ? AND whom_id = ?", whoID, whomID)
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
