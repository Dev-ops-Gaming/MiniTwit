package gorm_models

import (
	"fmt"
	"minitwit/models"

	"gorm.io/gorm"
)

type User struct {
	User_id  int `gorm:"primaryKey"`
	Username string
	Email    string
	Pw_hash  string
	//'Has many' relationship - message
	Messages []Message `gorm:"foreignKey:Author_id;references:User_id"`
	//Self-referential 'Many to Many' relationship - follow
	Followers []*User `gorm:"many2many:followers;foreignKey:User_id;joinForeignKey:Who_id;References:User_id;joinReferences:Whom_id;"`

	//https://gorm.io/docs/many_to_many.html
}

func GetUserByUsername(database *gorm.DB, username string) (*User, error) {
	var user User
	err := database.Table("Users").Where("username = ?", username).First(&user).Error
	if err != nil {
		fmt.Println("got error:")
		fmt.Println(err)
		return nil, err
	}
	return &user, nil

}

func GormUserToModelUser(gormUser *User) *models.User {
	convertedUser := models.User{ID: gormUser.User_id, Username: gormUser.Username, Email: gormUser.Email, PwHash: gormUser.Pw_hash}
	return &convertedUser

}

/*func GetUserByUsername(db *sql.DB, username string) (*User, error) {
	var user User
	err := db.QueryRow("SELECT user_id, username, email, pw_hash FROM user WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.Email, &user.PwHash)
	if err != nil {
		return nil, err
	}
	return &user, nil
}*/
