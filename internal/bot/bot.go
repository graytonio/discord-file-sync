package bot

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/graytonio/discord-git-sync/internal/commands"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Creates a new discord session with given token
func InitBot(token string, db *gorm.DB, guildID string) (*discordgo.Session, error) {
	s, err := discordgo.New(fmt.Sprintf("Bot %s", os.Getenv("DISCORD_BOT_TOKEN")))
	if err != nil {
		return nil, err
	}

	err = s.Open()
	if err != nil {
		return nil, err
	}

	err = addHandlers(s, db, guildID)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func addHandlers(s *discordgo.Session, db *gorm.DB, guildID string) error {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		logrus.Infof("Connected as %s", s.State.User.Username)
	})

	err := commands.CreateCommands(s, db, guildID)
	if err != nil {
		return err
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		commands.HandleCommand(s, i)
	})

	return nil
}
