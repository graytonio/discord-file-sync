package db

import (
	"time"

	"gorm.io/gorm"
)

func GetMessagesToUpdate(db *gorm.DB) ([]LinkedMessage, error) {
	messages := []LinkedMessage{}
	err := db.Where("next_update <= ?", time.Now()).Find(&messages).Error
	if err != nil {
		return nil, err
	}

	return messages, nil
}