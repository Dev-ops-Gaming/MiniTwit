package models

import (
	"database/sql"
)

type User struct {
	ID       int
	Username string
	Email    string
	Pwd      string //for register API
	PwHash   string
}

func GetUserByUsername(db *sql.DB, username string) (*User, error) {
	var user User
	err := db.QueryRow("SELECT user_id, username, email, pw_hash FROM user WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.Email, &user.PwHash)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
