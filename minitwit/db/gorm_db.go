package db

import (
	"minitwit/gorm_models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/mattn/go-sqlite3"
)

func Gorm_ConnectDB(database string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(database), &gorm.Config{
		// make gorm stop printing errors in terminal
		// gorm will print errors even if they are handled
		Logger: logger.Default.LogMode(logger.Silent),
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
	//defer db.Close() ??
}

//QueryTimeline

//QueryUserTimeline

//QueryPublicTimeline

//IsUserFollowing

// function must start w capital letter, or it isnt exported
/*func GormTest(database string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(database), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.AutoMigrate(&gorm_models.User{})
	if err != nil {
		panic("failed to migrate user table")
	}

	return db, nil
}*/
