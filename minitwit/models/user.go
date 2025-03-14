package models

import (
	"fmt"

	"gorm.io/gorm"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	User_id  int `gorm:"primaryKey"`
	Username string
	Email    string
	Pwd      string `gorm:"-"` //for register API
	Pw_hash  string
	//'Has many' relationship - message
	Messages []Message `gorm:"foreignKey:Author_id;references:User_id"`
	//Self-referential 'Many to Many' relationship - follow
	Followers []*User `gorm:"many2many:followers;foreignKey:User_id;joinForeignKey:Who_id;References:User_id;joinReferences:Whom_id;"`

	//https://gorm.io/docs/many_to_many.html
}

func GetUserByUsername(database *gorm.DB, username string) (*User, error) {
	var user User
	err := database.Table("users").Where("username = ?", username).First(&user).Error
	if err != nil {
		fmt.Println("got error:")
		fmt.Println(err)
		return nil, err
	}
	return &user, nil
}
