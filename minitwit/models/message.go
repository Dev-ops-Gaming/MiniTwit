package models

type Message struct {
	Message_id int `gorm:"primaryKey"`
	Author_id  uint
	Author     string `gorm:"-"` // ignore this field when write and read to db
	Email      string `gorm:"-"`
	Text       string
	Pub_date   int64
	PubDate    string `gorm:"-"`
	Flagged    int
}
