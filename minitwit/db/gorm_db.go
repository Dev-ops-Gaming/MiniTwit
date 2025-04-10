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

// ugly but temporary solution to be able to query messages with limit and order
type tempMessage struct {
	MessageID int    `gorm:"column:message_id"`
	AuthorID  uint   `gorm:"column:author_id"`
	Username  string `gorm:"column:username"`
	Email     string `gorm:"column:email"`
	Text      string `gorm:"column:text"`
	PubDate   int64  `gorm:"column:pub_date"`
}

// Helper function to convert intermediate messages to models.Message
func convertToMessages(messages []tempMessage) []models.Message {
	result := make([]models.Message, len(messages))
	for i, m := range messages {
		result[i] = models.Message{
			Message_id: m.MessageID,
			Author_id:  m.AuthorID,
			Author:     m.Username,
			Email:      m.Email,
			Text:       m.Text,
			Pub_date:   m.PubDate,
			PubDate:    utils.FormatTime(m.PubDate),
		}
	}
	return result
}

// flexible query function to query messages with where clause and args
// fits for all timeline queries
func queryMessages(db *gorm.DB, whereClause string, args ...interface{}) ([]models.Message, error) {
	var messages []tempMessage

	err := db.Table("messages").
		Select("messages.message_id, messages.author_id, users.username, users.email, messages.text, messages.pub_date").
		Joins("JOIN users ON messages.author_id = users.user_id").
		Where(whereClause, args...).
		Order("messages.pub_date DESC").
		Limit(PER_PAGE).
		Find(&messages).Error

	if err != nil {
		return nil, err
	}

	return convertToMessages(messages), nil
}

// Queries the timeline ("/")
func QueryTimeline(db *gorm.DB, userID int) ([]models.Message, error) {
	// Get list of whom user is following
	var followers []int
	db.Model(&models.Follower{}).Where("Who_id = ?", userID).Select("whom_id").Find(&followers)

	// Add current user to followers for the query
	followersWithUser := append(followers, userID)

	return queryMessages(db, "messages.flagged = 0 AND users.user_id IN ?", followersWithUser)
}

// Queries the user's timeline ("/<username>")
func QueryUserTimeline(db *gorm.DB, username string) ([]models.Message, error) {
	return queryMessages(db, "messages.flagged = 0 AND users.username = ?", username)
}

// Queries the public timeline ("/public")
func QueryPublicTimeline(db *gorm.DB) ([]models.Message, error) {
	return queryMessages(db, "messages.flagged = 0")
}

func IsUserFollowing(db *gorm.DB, whoID, whomID int) (bool, error) {
	var count int64
	err := db.Table("followers").Where("who_id = ? AND whom_id = ?", whoID, whomID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
