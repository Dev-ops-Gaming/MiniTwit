package db

import (
	"database/sql"
	"fmt"

	"minitwit/models"
	"minitwit/utils"

	_ "github.com/mattn/go-sqlite3"
)

var PER_PAGE = 30

func ConnectDB(database string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", database)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func QueryDB(db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func QueryTimeline(db *sql.DB, userID int) ([]models.Message, error) {
	rows, err := QueryDB(db, `
		select message.*, user.* 
		from message, user
        where message.flagged = 0 and message.author_id = user.user_id and (
            user.user_id = ? or
            user.user_id in (select whom_id from follower
                                    where who_id = ?))
		order by message.pub_date desc limit ?`, userID, userID, PER_PAGE)
	if err != nil {
		fmt.Println("Error in queryTimeline: ", err)
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		// Refactor later
		var pubDate int64
		var messageID, authorID, flagged, userID int
		var text, username, email, pwHash string

		err := rows.Scan(&messageID, &authorID, &text, &pubDate, &flagged, &userID, &username, &email, &pwHash)
		var m models.Message
		m = models.Message{ID: messageID, Author: username, Content: text, Email: email}
		m.PubDate = utils.FormatTime(pubDate) // Convert timestamp from UNIX to readable format
		if err != nil {
			fmt.Println("Error in queryTimeline: ", err)
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func QueryUserTimeline(db *sql.DB, username string) ([]models.Message, error) {
	rows, err := QueryDB(db, `
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

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		var pubDate int64
		err := rows.Scan(&m.ID, &m.Author, &m.Content, &pubDate, &m.Email)
		m.PubDate = utils.FormatTime(pubDate)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func QueryPublicTimeline(db *sql.DB) ([]models.Message, error) {
	rows, err := QueryDB(db, `
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

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		var pubDate int64
		err := rows.Scan(&m.ID, &m.Author, &m.Content, &pubDate, &m.Email)
		m.PubDate = utils.FormatTime(pubDate) // Convert timestamp from UNIX to readable format
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func IsUserFollowing(db *sql.DB, whoID, whomID int) (bool, error) {
	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM follower WHERE who_id = ? AND whom_id = ?", whoID, whomID)
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
