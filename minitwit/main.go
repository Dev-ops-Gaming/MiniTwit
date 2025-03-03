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

const GORMPATH = "./minitwit_gorm.db"

func main() {
	// Db logic
	database, err := db.ConnectDB(DBPATH)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer database.Close()

	// DB abstraction
	//this MUST be called, otherwise tests fail
	//seems grom cant read already existing database w/out migration stuff
	db.AutoMigrateDB(GORMPATH)
	gorm_db := db.Gorm_ConnectDB(GORMPATH)

	// Routes
	r := mux.NewRouter()
	//r.HandleFunc("/", handlers.TimelineHandler(database)).Methods("GET")
	r.HandleFunc("/", handlers.TimelineHandler(gorm_db)).Methods("GET")
	r.HandleFunc("/public", handlers.PublicTimelineHandler(gorm_db)).Methods("GET")
	r.HandleFunc("/register", handlers.RegisterHandler(gorm_db)).Methods("GET", "POST")
	r.HandleFunc("/login", handlers.LoginHandler(gorm_db)).Methods("GET", "POST")
	r.HandleFunc("/logout", handlers.LogoutHandler()).Methods("GET")
	//r.HandleFunc("/{username}/follow", handlers.FollowHandler(gorm_db)).Methods("GET", "POST")
	//handler below seems to get called when doing username/follow
	//gives problems with gorm rn
	r.HandleFunc("/{username}", handlers.UserTimelineHandler(gorm_db)).Methods("GET")
	r.HandleFunc("/{username}/follow", handlers.FollowHandler(gorm_db)).Methods("GET", "POST")
	r.HandleFunc("/{username}/unfollow", handlers.UnfollowHandler(gorm_db)).Methods("GET", "POST")
	r.HandleFunc("/add_message", handlers.AddMessageHandler(gorm_db)).Methods("POST")

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
