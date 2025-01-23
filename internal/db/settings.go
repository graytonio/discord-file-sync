package db

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Setting string

const (
	PageBreakEnabled Setting = "page-break"
)

var defaultSettings = map[Setting]GuildSetting{
	PageBreakEnabled: {
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Setting:   PageBreakEnabled,
		Enabled:   true,
	},
}

func GetGuildSetting(db *gorm.DB, guildID string, setting Setting) (*GuildSetting, error) {
	guildSetting := GuildSetting{}
	err := db.Where(&GuildSetting{GuildID: guildID, Setting: setting}).First(&guildSetting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			guildSetting = defaultSettings[setting]
			guildSetting.GuildID = guildID
			return &guildSetting, nil
		}

		return nil, err
	}

	return &guildSetting, nil
}
