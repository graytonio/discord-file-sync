package bot

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/graytonio/discord-git-sync/internal/commands"
	"github.com/sirupsen/logrus"
)

// Creates a new discord session with given token
func InitBot(token string, guildID string) (*discordgo.Session, error) {
	s, err := discordgo.New(fmt.Sprintf("Bot %s", os.Getenv("DISCORD_BOT_TOKEN")))
	if err != nil {
		return nil, err
	}

	err = s.Open()
	if err != nil {
		return nil, err
	}

	err = addHandlers(s, guildID)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func addHandlers(s *discordgo.Session, guildID string) error {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		logrus.Infof("Connected as %s", s.State.User.Username)
	})

	err := commands.CreateCommands(s, guildID)
	if err != nil {
		return err
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		commands.HandleCommand(s, i)
	})

	return nil
}
