package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

const (
	DATABASE   = "./minitwit.db"
	PER_PAGE   = 30
	SECRET_KEY = "development key"
)

var db *sql.DB
var templates = template.Must(template.ParseGlob("templates/*.html"))

func main() {
	db := connect_db() // Connect to the database
	defer db.Close()

	r := mux.NewRouter()
	r.HandleFunc("/", timeline)

	r.HandleFunc("/public", public_timeline)

	port := ":8080"
	log.Println("Server running on http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, r))
}

func connect_db() sql.DB {
	db, err := sql.Open("sqlite3", DATABASE)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	return *db
}

func query_db() {
	// TODO
}

func get_user_id() {
	// TODO
}

func format_datetime() {
	// TODO
}
func gravatar_url() {
	// TODO
}

func before_request() {} // mabIs also not need since we have db in as public variable

func timeline(w http.ResponseWriter, r *http.Request) {
	// TODO: we need to pass the data to renderTemplate
	renderTemplate(w, "test.html", nil)
}

func public_timeline(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func user_timeline() {
	// TODO
}

func follow_user() {
	// TODO
}

func add_message() {
	// TODO
}

func unfollow_user() {
	// TODO
}

func login() {
	// TODO
}

func register() {
	// TODO
}

func logout() {
	// TODO
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//aadd jinja stuff?

func testDb() {
	// Test the connection
	data, err := db.Query("select username from user")
	if err != nil {
		log.Fatalf("Failed to ping the database: %v", err)
	}

	for data.Next() {
		var user string
		err := data.Scan(&user)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(user)
	}

	fmt.Println("Connected to the database successfully!")
}
