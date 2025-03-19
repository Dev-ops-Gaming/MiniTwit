package db

import (
	"fmt"
	"minitwit/models"
	"minitwit/utils"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var PER_PAGE = 30

func Gorm_ConnectDB() *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_DBNAME"), os.Getenv("DB_PORT"), os.Getenv("DB_SSLMODE"), os.Getenv("DB_TIMEZONE"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// make gorm stop printing errors in terminal as otherwise
		// gorm will print errors even if they are handled
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		panic("failed to connect database")
	}

	return db
}

func AutoMigrateDB() {
	// Creates/Connects to the database tables
	db := Gorm_ConnectDB()
	err := db.AutoMigrate(&models.User{}, &models.Message{})
	if err != nil {
		panic("failed to migrate database tables")
	}
}

func GormGetUserId(db *gorm.DB, username string) (int, error) {
	user := models.User{}
	// Get first matched record
	result := db.Select("user_id").Where("username = ?", username).First(&user)
	// returns 0, err if nothing found
	return user.User_id, result.Error
}

// helper function to convert users messages to messages
func convertUserMessagesToMessages(users []models.User) []models.Message {
	var messages []models.Message
	for _, user := range users {
		for _, message := range user.Messages {
			messages = append(messages, models.Message{
				Message_id: message.Message_id,
				Author:     user.Username,
				Text:       message.Text,
				Email:      user.Email,
				PubDate:    utils.FormatTime(message.Pub_date),
			})
		}
	}
	return messages
}

// helper function for preloading messages that are not flagged and ordered by pub_date
func getMessagesQuery(db *gorm.DB) *gorm.DB {
	return db.Preload("Messages", "flagged = 0", func(database *gorm.DB) *gorm.DB {
		return database.Order("pub_date DESC")
	})
}

// Queries the timeline ("/")
func QueryTimeline(db *gorm.DB, userID int) ([]models.Message, error) {
	// Get list of whom user is following
	var followers []int
	db.Model(&models.Follower{}).Where("Who_id = ?", userID).Select("whom_id").Find(&followers)

	// Get all messages made by either current user or people they're following
	var users []models.User
	getMessagesQuery(db).
		Table("users").
		Where("user_id = ? OR user_id IN ?", userID, followers).
		Limit(PER_PAGE).
		Find(&users)

	return convertUserMessagesToMessages(users), nil
}

// Queries the user's timeline ("/<username>")
func QueryUserTimeline(db *gorm.DB, username string) ([]models.Message, error) {
	var users []models.User
	getMessagesQuery(db).
		Where("Username = ?", username).
		Limit(PER_PAGE).
		Find(&users)

	return convertUserMessagesToMessages(users), nil
}

// Queries the public timeline ("/public")
func QueryPublicTimeline(db *gorm.DB) ([]models.Message, error) {
	var users []models.User
	getMessagesQuery(db).
		Limit(PER_PAGE).
		Find(&users)

	return convertUserMessagesToMessages(users), nil
}

func IsUserFollowing(db *gorm.DB, whoID, whomID int) (bool, error) {
	var count int64
	err := db.Table("followers").Where("who_id = ? AND whom_id = ?", whoID, whomID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
