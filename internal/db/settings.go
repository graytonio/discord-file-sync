package db

import "gorm.io/gorm"

type Setting string

const (
	PageBreakEnabled Setting = "page-break"
)

func GetGuildSetting(db *gorm.DB, guildID string, setting Setting) (*GuildSetting, error) {
	guildSetting := GuildSetting{Enabled: false}
	err := db.Where(&GuildSetting{GuildID: guildID, Setting: setting}).First(&guildSetting).Error
	if err != nil { // TODO Allow for default setting if record not found
	  return nil, err
	}

	return &guildSetting, nil
} 