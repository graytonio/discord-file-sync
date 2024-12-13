package commands

import (
	"fmt"
	"net/url"

	"github.com/bwmarrin/discordgo"
	"github.com/graytonio/discord-git-sync/internal/manager"
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
		Name:        "link-message",
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
		log := logrus.WithFields(logrus.Fields{
			"command":        "link",
			"user":           i.Member.User.ID,
			"guild":          i.GuildID,
			"interaction_id": i.ID,
		})
		log.Info("linking new webpage")

		opts := commandOptionsToMap(i.ApplicationCommandData().Options)

		parsedURL, err := url.ParseRequestURI(opts["link"].StringValue())
		if err != nil {
			metrics.CommandsFailed.With(prometheus.Labels{"command": "link"}).Inc()
			log.WithError(err).Error("invalid link url")
			l.sendErrorResponse(s, i, log, err)
			return
		}

		msgID, err := manager.CreateNewLinkedMessage(log, s, l.db, parsedURL, i.GuildID, i.ChannelID)
		if err != nil {
			metrics.CommandsFailed.With(prometheus.Labels{"command": "link"}).Inc()
			log.WithError(err).Error("cloud not link new message")
			l.sendErrorResponse(s, i, log, err)
			return
		}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: fmt.Sprintf("New git linked message created: https://discord.com/channels/%s/%s/%s", i.GuildID, i.ChannelID, msgID),
			},
		})
		if err != nil {
			metrics.CommandsFailed.With(prometheus.Labels{"command": "link"}).Inc()
			log.WithError(err).Error("could not respond to user")
			return
		}
		metrics.CommandsServed.With(prometheus.Labels{"command": "link"}).Inc()
	}
}

func (l *LinkCommand) sendErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, log *logrus.Entry, err error) {
	metrics.CommandsFailed.With(prometheus.Labels{"command": "link"}).Inc()

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("There was a problem creating your linked git message: %s", err.Error()),
		},
	})
	if err != nil {
		log.WithError(err).Error("could not respond to user")
	}
}
