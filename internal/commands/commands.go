package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type SlashCommand interface {
	GetDefinition() *discordgo.ApplicationCommand
	GetHandler() func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){}

func buildCommands(db *gorm.DB) []SlashCommand {
	return []SlashCommand{
		&LinkCommand{
			db: db,
		},
		&UpdateCommand{
			db: db,
		},
	}
}

func CreateCommands(s *discordgo.Session, db *gorm.DB, guildID string) error {
	for _, v := range buildCommands(db) {
		logrus.WithField("command", v.GetDefinition().Name).Debug("creating command")
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, v.GetDefinition())
		if err != nil {
			return err
		}

		commandHandlers[cmd.Name] = v.GetHandler()
	}

	return nil
}

func HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
		h(s, i)
	}
}
