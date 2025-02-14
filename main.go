package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
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

// currently unused
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
	r.HandleFunc("/register", registerHandler).Methods("GET", "POST")
	r.HandleFunc("/login", loginHandler).Methods("GET", "POST")
	r.HandleFunc("/logout", logoutHandler).Methods("GET")
	r.HandleFunc("/{username}", userTimelineHandler).Methods("GET")

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// General functions

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

// Handlers

func timelineHandler(w http.ResponseWriter, r *http.Request) {
	messages, err := queryTimeline()
	if err != nil {
		http.Error(w, "Failed to load timeline", http.StatusInternalServerError)
		return
	}

	data := struct {
		Messages []Message
	}{
		Messages: messages,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func userTimelineHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]
	messages, err := queryUserTimeline(username)
	if err != nil {
		http.Error(w, "Failed to load user timeline", http.StatusInternalServerError)
		return
	}

	data := struct {
		Messages []Message
	}{
		Messages: messages,
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
		http.Redirect(w, r, "/", http.StatusFound)
		//TODO: REMEMBER TO ADD COOKIE
	}
}

//Query functions

func queryTimeline() ([]Message, error) {
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
		err = store.Save(r, w)

		// Redirect to timeline
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	store, _ := store.Get(r, "minitwit-session")
	if store.Values["user_id"] == nil {
		http.Error(w, "You are not logged in", http.StatusBadRequest)
		return
	}
	store.Options.MaxAge = -1 // Clear session
	store.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}
