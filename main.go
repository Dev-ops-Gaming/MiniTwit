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
	"strings"
	"time"

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
	r.HandleFunc("/", timeline).Methods("GET")
	r.HandleFunc("/public", public_timeline).Methods("GET")
	r.HandleFunc("/{username}", user_timeline).Methods("GET")
	r.HandleFunc("/{username}/follow", follow_user).Methods("GET")
	r.HandleFunc("/{username}/unfollow", unfollow_user).Methods("GET")
	r.HandleFunc("/add_message", add_message).Methods("POST")
	r.HandleFunc("/login", login).Methods("GET", "POST")
	r.HandleFunc("/register", register).Methods("GET", "POST")
	r.HandleFunc("/logout", logout).Methods("GET")

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

func init_db() {
	// Creates the database tables.
	db := connect_db()
	file, err := os.ReadFile("schema.sql")
	if err != nil {
		log.Fatalf("Failed to read sql script: %v", err)
	}
	fileAsString := string(file)
	_, err = db.Exec(fileAsString)
	if err != nil {
		log.Fatalf("Failed to create the database tables: %v", err)
	}
	//orig code uses .commit(). Are our changes commited?
	//it does not insert values. where did we get those?
}

// 'one=false', Golang does not support optional/default parameters
// return type interface{}/any as two diff maps can be returned
// https://stackoverflow.com/questions/35657362/how-to-return-dynamic-type-struct-in-golang
// IMPORTANT - must use switch case to handle return from this method! Check testDB() for examples!
func query_db(query string, args []any, one bool) any {
	// Queries the database and returns a list of dictionaries.
	db := connect_db()
	cur, err := db.Query(query, args...)
	if err != nil {
		log.Fatalf("Failed query the database: %v", err)
	}

	var rv = map[int]map[string]any{}
	var i int = 0
	//for every row
	for cur.Next() {
		//make map[col]value
		rv[i] = make(map[string]any)
		var val any
		//for each col, insert value in map[col]value
		cols, _ := cur.Columns()
		for _, col := range cols {
			err := cur.Scan(&val)
			if err != nil {
				fmt.Println(err)
			}
			rv[i][col] = val
		}
		i += 1
	}
	//fmt.Println(rv[0]["username"])
	if one {
		return rv[0]
	} else {
		return rv
	}
}

// either return pair or 'any'?
func get_user_id(username string) (int, any) {
	// Convenience method to look up the id for a username.
	db := connect_db()
	var userId int
	// use .QueryRow to handle possible empty results
	if err := db.QueryRow("select user_id from user where username = ?", username).Scan(&userId); err == sql.ErrNoRows {
		//empty result
		return 0, err
	} else {
		//got smth
		return userId, nil
	}
}

func format_datetime(timestamp int64) string {
	// Format the date and time
	return time.Unix(timestamp, 0).Format("02-01-2006 15:04:05") // dd-mm-yyyy hh:mm:ss
}
func gravatar_url(email string, size int) string {
	// Return the gravatar image for the given email address.
	hash := md5.New()
	hash.Write([]byte(strings.ToLower(email)))
	return fmt.Sprintf("http://www.gravatar.com/avatar/%s?d=identicon&s=%d", hex.EncodeToString(hash.Sum(nil)), size)
}

func before_request() {} // mabIs also not need since we have db in as public variable

func timeline(w http.ResponseWriter, r *http.Request) {
	// TODO: we need to pass the data to renderTemplate
	renderTemplate(w, "test.html", nil)
}

func public_timeline(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func user_timeline(w http.ResponseWriter, r *http.Request) {
	// TODO: not done yet. as of now, we can get the username from the URL
	vars := mux.Vars(r) // gets the variables from the URL
	username := vars["username"]
	println("User timeline of " + username)
}

func follow_user(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func add_message(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func unfollow_user(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func login(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func register(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func logout(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//add jinja stuff?

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

	// --- Example for query_db() ---
	args := []any{} //empty collection of 'any'
	var res = query_db("SELECT username FROM user", args, true)

	//type conversion
	// https://stackoverflow.com/questions/47496040/type-interface-does-not-support-indexing-in-golang

	// switch cases to handle 'any' return type from query_db()
	switch res := res.(type) {
	case map[int]map[string]any:
		fmt.Println(res)
	case map[string]any:
		fmt.Println("got username:")
		fmt.Println(res["username"])
	}
}
