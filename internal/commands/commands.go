package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

type SlashCommand interface {
	GetDefinition() *discordgo.ApplicationCommand
	GetHandler() func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

var commands = []SlashCommand{
	&LinkCommand{},
}

var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){}

func CreateCommands(s *discordgo.Session, guildID string) error {
	for _, v := range commands {
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
