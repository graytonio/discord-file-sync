package db

import (
	"database/sql/driver"
	"errors"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type MessageChain []string

func (mc *MessageChain) Scan(src any) error {
	bytes, ok := src.([]byte)
	if !ok {
		return errors.New("src value cannot cast to []byte")
	}
	*mc = strings.Split(string(bytes), ",")
	return nil
}

func (mc MessageChain) Value() (driver.Value, error) {
	if len(mc) == 0 {
	 return nil, nil
	}
	
	return strings.Join(mc, ","), nil
   }

type LinkedMessage struct {
	gorm.Model
	GuildID      string
	ChannelID    string
	MessageID    string       `gorm:"index"`
	MessageChain MessageChain `gorm:"type:TEXT"`
	LinkedPage   datatypes.URL
}

type GuildSetting struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	GuildID   string         `gorm:"primaryKey"`
	Setting   Setting        `gorm:"primaryKey"`
	Enabled   bool
}

func InitDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&LinkedMessage{}, &GuildSetting{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
