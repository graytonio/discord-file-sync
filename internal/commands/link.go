package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/graytonio/discord-git-sync/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type LinkCommand struct {
	db *gorm.DB
}

var _ SlashCommand = &LinkCommand{}

// GetDefinition implements SlashCommand.
func (l *LinkCommand) GetDefinition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "link",
		Description: "Link a github file to a new discord message",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "link",
				Description: "Raw Github file link to new discord message",
				Required:    true,
				Type:        discordgo.ApplicationCommandOptionString,
			},
		},
	}
}

// GetHandler implements SlashCommand.
func (l *LinkCommand) GetHandler() func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		logrus.WithField("user", i.Member.User.ID).WithField("guild", i.GuildID).Info("Creating new git link")

		// TODO Fetch data from link

		msg, err := s.ChannelMessageSend(i.ChannelID, "New GitHub Message!")
		if err != nil {
			metrics.CommandsFailed.With(prometheus.Labels{"command": "link"}).Inc()
			logrus.WithError(err).Error("could not send new linked message")
			err = l.SendErrorResponse(s, i)
			if err != nil {
				logrus.WithError(err).Error("could not respond to user")
			}
		}

		// TODO Store message link in db

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: fmt.Sprintf("New git linked message created: https://discord.com/channels/%s/%s/%s", i.GuildID, i.ChannelID, msg.ID),
			},
		})
		if err != nil {
			metrics.CommandsFailed.With(prometheus.Labels{"command": "link"}).Inc()
			logrus.WithError(err).Error("could not respond to user")
		}
		metrics.CommandsServed.With(prometheus.Labels{"command": "link"}).Inc()
	}
}

func (l *LinkCommand) SendErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "There was a problem creating your linked git message",
		},
	})
}