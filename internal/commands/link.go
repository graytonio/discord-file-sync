package commands

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/bwmarrin/discordgo"
	"github.com/graytonio/discord-git-sync/internal/db"
	"github.com/graytonio/discord-git-sync/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
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
			log.WithError(err).Error("invalid link url")
			l.sendErrorResponse(s, i, log, err)
			return
		}

		content, err := fetchPage(log, parsedURL)
		if err != nil {
			log.WithError(err).Error("could not get content to link")
			l.sendErrorResponse(s, i, log, err)
			return
		}

		setting := db.GuildSetting{Enabled: false} // Default to False
		err = l.db.Where(&db.GuildSetting{GuildID: i.GuildID, Setting: db.PageBreakEnabled}).First(&setting).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.WithError(err).Error("could not fetch guild settings")
			l.sendErrorResponse(s, i, log, err)
			return
		}

		embeds, err := buildEmbedContents(content, parsedURL, setting.Enabled)
		if err != nil {
			log.WithError(err).Error("could process link content")
			l.sendErrorResponse(s, i, log, err)
			return
		}

		msgIDs := []string{}
		for _, emb := range embeds {
			msg, err := s.ChannelMessageSendEmbed(i.ChannelID, emb)
			if err != nil {
				log.WithError(err).Error("could not send new linked message")
				l.sendErrorResponse(s, i, log, err)
				return
			}

			msgIDs = append(msgIDs, msg.ID)
		}

		err = l.db.Create(&db.LinkedMessage{
			GuildID:    i.GuildID,
			ChannelID:  i.ChannelID,
			MessageID:  msgIDs[0],
			MessageChain: msgIDs,
			LinkedPage: datatypes.URL(*parsedURL),
		}).Error
		if err != nil {
			log.WithError(err).Error("could not save message link")
			l.sendErrorResponse(s, i, log, err)
			return
		}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: fmt.Sprintf("New git linked message created: https://discord.com/channels/%s/%s/%s", i.GuildID, i.ChannelID, msgIDs[0]),
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
