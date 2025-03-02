package gorm_models

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
