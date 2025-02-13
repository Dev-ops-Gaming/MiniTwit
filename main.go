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

func main() {
	// Db logic
	db = connectDB()
	defer db.Close()

	// Routes
	r := mux.NewRouter()
	r.HandleFunc("/", timelineHandler).Methods("GET")

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
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

func timelineHandler(w http.ResponseWriter, r *http.Request) {
	messages, err := queryTimeline()
	if err != nil {
		http.Error(w, "Failed to load timeline", http.StatusInternalServerError)
		fmt.Printf("Failed to load timeline: %v\n", err)
		return
	}

	data := struct {
		Messages []Message
	}{
		Messages: messages,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		fmt.Printf("Failed to render template: %v\n", err)
	}
}

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
