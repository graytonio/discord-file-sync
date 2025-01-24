package db

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Setting string

const (
	PageBreakEnabled Setting = "page-break"
	MessageAutoUpdateRate Setting = "update-rate"
)

var defaultSettings = map[Setting]GuildSetting{
	PageBreakEnabled: {
		Setting:   PageBreakEnabled,
		Enabled:   true,
	},
	MessageAutoUpdateRate: {
		Setting: MessageAutoUpdateRate,
		DurationValue: time.Hour,
	},
}

func GetGuildSetting(db *gorm.DB, guildID string, setting Setting) (*GuildSetting, error) {
	guildSetting := GuildSetting{}
	err := db.Where(&GuildSetting{GuildID: guildID, Setting: setting}).First(&guildSetting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return setDefaultGuildSetting(db, guildID, setting)
		}

		return nil, err
	}

	return &guildSetting, nil
}

func setDefaultGuildSetting(db *gorm.DB, guildID string, setting Setting) (*GuildSetting, error) {
	defaultValue := defaultSettings[setting]
	defaultValue.GuildID = guildID
	err := db.Create(&defaultValue).Error
	if err != nil {
	  return nil, err
	}

	return &defaultValue, nil
}
