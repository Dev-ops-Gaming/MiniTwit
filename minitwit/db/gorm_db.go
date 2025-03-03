package db

import (
	"fmt"
	"minitwit/gorm_models"
	"minitwit/models"
	"minitwit/utils"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "github.com/mattn/go-sqlite3"
)

func Gorm_ConnectDB(database string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(database), &gorm.Config{
		// make gorm stop printing errors in terminal
		// gorm will print errors even if they are handled
		//Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic("failed to connect database")
	}

	return db
}

func AutoMigrateDB(database string) {
	// Creates the database tables
	db := Gorm_ConnectDB(database)
	err := db.AutoMigrate(&gorm_models.User{}, &gorm_models.Message{}) //, &gorm_models.Follower{})
	if err != nil {
		panic("failed to migrate database tables")
	}
	//according to gorm documentation, doesnt seem like .Close is needed
	//defer db.Close() ??
}

func GormGetUserId(db *gorm.DB, username string) (int, error) {
	// Get first matched record
	user := gorm_models.User{}
	result := db.Select("user_id").Where("username = ?", username).First(&user)
	return user.User_id, result.Error
}

// QueryTimeline
func QueryTimeline(db *gorm.DB, userID int) ([]models.Message, error) {
	//get list of who user is following
	var followers []int
	db.Model(&gorm_models.Follower{}).Where("Who_id = ?", userID).Select("whom_id").Find(&followers)

	//get all messages made by either current user or people they're following
	var users []gorm_models.User
	db.Table("Users").Where("user_id = ? OR user_id IN ?", userID, followers).Preload("Messages", "flagged = 0", func(database *gorm.DB) *gorm.DB {
		db := database.Order("pub_date DESC")
		return db
	}).Limit(PER_PAGE).Find(&users)

	var messages []models.Message
	for _, user := range users {
		for _, message := range user.Messages {
			//convert gorm_models.message to models.message
			m := models.Message{ID: message.Message_id, Author: user.Username, Content: message.Text, Email: user.Email}
			m.PubDate = utils.FormatTime(message.Pub_date) // Convert timestamp from UNIX to readable format
			messages = append(messages, m)
		}
	}
	return messages, nil
}

// QueryUserTimeline
func QueryUserTimeline(db *gorm.DB, username string) ([]models.Message, error) {
	var users []gorm_models.User
	db.Model(&gorm_models.User{}).Preload("Messages", "flagged = 0", func(database *gorm.DB) *gorm.DB {
		db := database.Order("pub_date DESC")
		return db
	}).Where("Username = ?", username).Limit(PER_PAGE).Find(&users)

	var messages []models.Message
	for _, user := range users {
		for _, message := range user.Messages {
			//convert gorm_models.message to models.message
			m := models.Message{ID: message.Message_id, Author: user.Username, Content: message.Text, Email: user.Email}
			m.PubDate = utils.FormatTime(message.Pub_date) // Convert timestamp from UNIX to readable format
			messages = append(messages, m)
		}
	}
	return messages, nil
}

func QueryPublicTimeline(db *gorm.DB) ([]models.Message, error) {
	var users []gorm_models.User
	db.Model(&gorm_models.User{}).Preload("Messages", "flagged = 0", func(database *gorm.DB) *gorm.DB {
		db := database.Order("pub_date DESC")
		return db
	}).Limit(PER_PAGE).Find(&users)

	var messages []models.Message
	for _, user := range users {
		for _, message := range user.Messages {
			//convert gorm_models.message to models.message
			m := models.Message{ID: message.Message_id, Author: user.Username, Content: message.Text, Email: user.Email}
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
