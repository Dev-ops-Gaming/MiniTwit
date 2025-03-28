package main

import (
	"fmt"
	"log"
	"net/http"

	"minitwit/db"
	"minitwit/handlers"
	"minitwit/middleware"

	"github.com/gorilla/mux"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// DB abstraction
	//this MUST be called, otherwise tests fail
	//seems grom cant read already existing database w/out migration stuff
	db.AutoMigrateDB()
	gorm_db := db.Gorm_ConnectDB()

	// Routes
	r := mux.NewRouter()

	// Middleware
	r.Use(middleware.PrometheusMiddleware)

	// expose metrics
	r.Handle("/metrics", promhttp.Handler())

	// general routes
	r.HandleFunc("/", handlers.TimelineHandler(gorm_db)).Methods("GET")
	r.HandleFunc("/public", handlers.PublicTimelineHandler(gorm_db)).Methods("GET")
	r.HandleFunc("/register", handlers.RegisterHandler(gorm_db)).Methods("GET", "POST")
	r.HandleFunc("/login", handlers.LoginHandler(gorm_db)).Methods("GET", "POST")
	r.HandleFunc("/logout", handlers.LogoutHandler()).Methods("GET")
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
