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
	//We only create tables Users and Messages
	//Table Followers will be created automatically - see gorm_models.User
	err := db.AutoMigrate(&models.User{}, &models.Message{})
	if err != nil {
		panic("failed to migrate database tables")
	}
	//according to gorm documentation, doesnt seem like .Close is needed
	//defer db.Close() ??
}

func GormGetUserId(db *gorm.DB, username string) (int, error) {
	user := models.User{}
	// Get first matched record
	result := db.Select("user_id").Where("username = ?", username).First(&user)
	// returns 0, err if nothing found
	return user.User_id, result.Error
}

func QueryTimeline(db *gorm.DB, userID int) ([]models.Message, error) {
	//get list of whom user is following
	var followers []int
	db.Model(&models.Follower{}).Where("Who_id = ?", userID).Select("whom_id").Find(&followers)

	//get all messages made by either current user or people they're following
	var users []models.User
	db.Table("users").Where("user_id = ? OR user_id IN ?", userID, followers).Preload("Messages", "flagged = 0", func(database *gorm.DB) *gorm.DB {
		db := database.Order("pub_date DESC")
		return db
	}).Limit(PER_PAGE).Find(&users)

	var messages []models.Message
	for _, user := range users {
		for _, message := range user.Messages {
			//convert gorm_models.message to models.message
			m := models.Message{Message_id: message.Message_id, Author: user.Username, Text: message.Text, Email: user.Email}
			m.PubDate = utils.FormatTime(message.Pub_date) // Convert timestamp from UNIX to readable format
			messages = append(messages, m)
		}
	}
	return messages, nil
}

func QueryUserTimeline(db *gorm.DB, username string) ([]models.Message, error) {
	var users []models.User
	db.Model(&models.User{}).Preload("Messages", "flagged = 0", func(database *gorm.DB) *gorm.DB {
		db := database.Order("pub_date DESC")
		return db
	}).Where("Username = ?", username).Limit(PER_PAGE).Find(&users)

	var messages []models.Message
	for _, user := range users {
		for _, message := range user.Messages {
			//convert gorm_models.message to models.message
			m := models.Message{Message_id: message.Message_id, Author: user.Username, Text: message.Text, Email: user.Email}
			m.PubDate = utils.FormatTime(message.Pub_date) // Convert timestamp from UNIX to readable format
			messages = append(messages, m)
		}
	}
	return messages, nil
}

func QueryPublicTimeline(db *gorm.DB) ([]models.Message, error) {
	var users []models.User
	db.Model(&models.User{}).Preload("Messages", "flagged = 0", func(database *gorm.DB) *gorm.DB {
		db := database.Order("pub_date DESC")
		return db
	}).Limit(PER_PAGE).Find(&users)

	var messages []models.Message
	for _, user := range users {
		for _, message := range user.Messages {
			//convert gorm_models.message to models.message
			m := models.Message{Message_id: message.Message_id, Author: user.Username, Text: message.Text, Email: user.Email}
			m.PubDate = utils.FormatTime(message.Pub_date) // Convert timestamp from UNIX to readable format
			messages = append(messages, m)

			//Check message_id! bc of this old query:
			//SELECT message.author_id, user.username, message.text, message.pub_date, user.email
			//and this old code. Seems they put author_id in Message.ID instead of message_id??
			//err := rows.Scan(&m.ID, &m.Author, &m.Content, &pubDate, &m.Email)
		}
	}
	return messages, nil
}

func IsUserFollowing(db *gorm.DB, whoID, whomID int) (bool, error) {
	var count int64
	err := db.Table("Followers").Where("who_id = ? AND whom_id = ?", whoID, whomID).Count(&count).Error
	if err != nil {
		fmt.Println("got error when check follow: ")
		fmt.Println(err)
		return false, err
	}
	return count > 0, nil
}
