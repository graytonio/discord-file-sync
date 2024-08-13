package db

import (
	"gorm.io/datatypes"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type LinkedMessage struct {
	gorm.Model
	GuildID    string
	ChannelID  string
	MessageID  string `gorm:"index"`
	LinkedPage datatypes.URL
}

func InitDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
	  return nil, err
	}

	err = db.AutoMigrate(&LinkedMessage{})
	if err != nil {
	  return nil, err
	}

	return db, nil
}