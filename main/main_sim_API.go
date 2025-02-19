package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

//call init_db() - might not need this

func not_req_from_simulator(r *http.Request) []byte {
	from_simulator := r.Header.Get("Authorization")
	var err string
	if from_simulator != "Basic c2ltdWxhdG9yOnN1cGVyX3NhZmUh" {
		err = "You are not authorized to use this resource!"
	}
	jsonMap := map[string]any{
		"status":    403,
		"error_msg": err,
	}
	jsonSstring, erro := json.Marshal(jsonMap)
	if erro != nil {
		println(erro)
	}
	return jsonSstring
}

func get_user_id(username string) (int, any) {
	// Convenience method to look up the id for a username.
	db := connectDB()
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

//before & after request () ?? - maybe not needed

// update_latest(request: request):
func update_latest(r *http.Request) {
	// Get arg value associated with 'latest' & convert to int
	//parsed_command_id, err := strconv.Atoi(r.FormValue("latest"))
	parsed_command_id := r.FormValue("latest")

	//var f *os.File
	if parsed_command_id != "-1" || parsed_command_id != "" {
		f, err := os.OpenFile("./latest_processed_sim_action_id.txt", os.O_WRONLY, os.ModeAppend)
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

// get the latest value
func get_latest(w http.ResponseWriter, r *http.Request) {
	file, err := os.ReadFile("./latest_processed_sim_action_id.txt")
	if err != nil {
		log.Fatalf("Failed to read latest_id file: %v", err)
	}
	//convert file content to string, then int
	fileAsString := string(file)
	fileID, err := strconv.Atoi(fileAsString)
	if err != nil {
		log.Fatalf("Failed to convert file string to int: %v", err)
	}

	jsonMap := map[string]any{
		"latest": fileID,
	}
	//convert map to json object and return it
	j, err := json.Marshal(jsonMap)
	if err != nil {
		log.Fatalf("Failed to convert file string to int: %v", err)
	}
	w.Write(j)
}

// register POST
func register(r *http.Request) ([]byte, int) {
	update_latest(r)

	//must decode into struct bc data sent as json, which golang bitches abt
	d := json.NewDecoder(r.Body)
	var t User
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
		} else if id, err := get_user_id(t.Username); err == nil || id != 0 { //userid starts from 1 in db
			erro = "The username is already taken"
			fmt.Println(id)
		} else {
			// hash the password
			hash := md5.New()
			hash.Write([]byte(r.Form.Get("pwd")))
			pwHash := hex.EncodeToString(hash.Sum(nil))

			// insert the user into the database
			_, err := db.Exec("INSERT INTO user (username, email, pw_hash) VALUES (?, ?, ?)", r.Form.Get("username"), r.Form.Get("email"), pwHash)
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
		return jsonSstring, 400
	} else {
		return json.RawMessage(""), 204
	}

}

//messages GET

//messages_per_user GET POST

//follow GET POST
