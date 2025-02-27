package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"minitwit/db"
	"minitwit/handlers"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

const DBPATH = "./minitwit.db"

var database *sql.DB

func main() {
	// Db logic
	database, err := db.ConnectDB(DBPATH)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer database.Close()

	// Routes
	r := mux.NewRouter()
	r.HandleFunc("/", handlers.TimelineHandler(database)).Methods("GET")
	r.HandleFunc("/public", handlers.PublicTimelineHandler(database)).Methods("GET")
	r.HandleFunc("/register", handlers.RegisterHandler(database)).Methods("GET", "POST")
	r.HandleFunc("/login", handlers.LoginHandler(database)).Methods("GET", "POST")
	r.HandleFunc("/logout", handlers.LogoutHandler()).Methods("GET")
	r.HandleFunc("/{username}", handlers.UserTimelineHandler(database)).Methods("GET")
	r.HandleFunc("/{username}/follow", handlers.FollowHandler(database)).Methods("GET", "POST")
	r.HandleFunc("/{username}/unfollow", handlers.UnfollowHandler(database)).Methods("GET", "POST")
	r.HandleFunc("/add_message", handlers.AddMessageHandler(database)).Methods("POST")

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func InitDB() {
	// Creates the database tables
	db, err := db.ConnectDB(DBPATH)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	file, err := os.ReadFile("schema.sql")
	if err != nil {
		log.Fatalf("Failed to read sql script: %v", err)
	}
	fileAsString := string(file)
	_, err = db.Exec(fileAsString)
	if err != nil {
		log.Fatalf("Failed to create the database tables: %v", err)
	}
}
