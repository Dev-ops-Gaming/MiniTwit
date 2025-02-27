package models

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID       int
	Username string
	Email    string
	PwHash   string
}

func GetUserByUsername(db *sql.DB, username string) (*User, error) {
	var user User
	err := db.QueryRow("SELECT id, username, email, pw_hash FROM user WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.Email, &user.PwHash)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
