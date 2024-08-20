package commands

import (
	"errors"
	"fmt"
	"net/url"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/graytonio/discord-git-sync/internal/db"
	"github.com/graytonio/discord-git-sync/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UpdateCommand struct {
	db *gorm.DB
}

var _ SlashCommand = &UpdateCommand{}

// GetDefinition implements SlashCommand.
func (u *UpdateCommand) GetDefinition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "update-message",
		Description: "Update a linked message with the latest content",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "message",
				Description: "Link to message to update",
				Required:    true,
				Type:        discordgo.ApplicationCommandOptionString,
			},
		},
	}
}

// GetHandler implements SlashCommand.
func (u *UpdateCommand) GetHandler() func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		log := logrus.WithFields(logrus.Fields{
			"command":        "update",
			"user":           i.Member.User.ID,
			"guild":          i.GuildID,
			"interaction_id": i.ID,
		})
		log.Info("updating linked message")

		opts := commandOptionsToMap(i.ApplicationCommandData().Options)

		messageID := stripDiscordLinkMessageID(opts["message"].StringValue())

		linkedMessage := db.LinkedMessage{}
		err := u.db.Where(db.LinkedMessage{
			MessageID: messageID,
		}).First(&linkedMessage).Error
		if err != nil {
			log.WithError(err).Error("could not find linked message")
			u.sendErrorResponse(s, i, log, err)
			return
		}

		content, err := fetchPage(log, (*url.URL)(&linkedMessage.LinkedPage))
		if err != nil {
			log.WithError(err).Error("could not get content to link")
			u.sendErrorResponse(s, i, log, err)
			return
		}

		setting := db.GuildSetting{Enabled: false} // Default to False
		err = u.db.Where(&db.GuildSetting{GuildID: i.GuildID, Setting: db.PageBreakEnabled}).First(&setting).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.WithError(err).Error("could not fetch guild settings")
			u.sendErrorResponse(s, i, log, err)
			return
		}

		embeds, err := buildEmbedContents(content, (*url.URL)(&linkedMessage.LinkedPage), setting.Enabled)
		if err != nil {
			log.WithError(err).Error("could process link content")
			u.sendErrorResponse(s, i, log, err)
			return
		}

		msgIds := []string{}
		for j, e := range embeds {
			var msg *discordgo.Message
			if j >= len(linkedMessage.MessageChain) {
				msg, err = s.ChannelMessageSendEmbed(i.ChannelID, e)
				if err != nil {
					log.WithError(err).Error("could not send new linked message")
					u.sendErrorResponse(s, i, log, err)
					return
				}

			} else {
				msg, err = s.ChannelMessageEditEmbed(linkedMessage.ChannelID, linkedMessage.MessageChain[j], e)
				if err != nil {
					log.WithError(err).Error("could not update message content")
					u.sendErrorResponse(s, i, log, err)
					return
				}
			}

			msgIds = append(msgIds, msg.ID)
		}

		// Trims not needed messages
		for _, m := range linkedMessage.MessageChain {
			if slices.Contains(msgIds, m) {
				continue
			}

			err = s.ChannelMessageDelete(linkedMessage.ChannelID, m)
				if err != nil {
					log.WithError(err).Error("could not update message content")
					u.sendErrorResponse(s, i, log, err)
					return
				}
		}

		linkedMessage.MessageChain = msgIds
		linkedMessage.MessageID = msgIds[0]
		err = u.db.Save(&linkedMessage).Error
		if err != nil {
			log.WithError(err).Error("could not update message db record")
			u.sendErrorResponse(s, i, log, err)
			return
		}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: fmt.Sprintf("Message content updated: https://discord.com/channels/%s/%s/%s", linkedMessage.GuildID, linkedMessage.ChannelID, msgIds[0]),
			},
		})
		if err != nil {
			metrics.CommandsFailed.With(prometheus.Labels{"command": "update"}).Inc()
			log.WithError(err).Error("could not respond to user")
			return
		}
		metrics.CommandsServed.With(prometheus.Labels{"command": "update"}).Inc()
	}
}

func (u *UpdateCommand) sendErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, log *logrus.Entry, err error) {
	metrics.CommandsFailed.With(prometheus.Labels{"command": "update"}).Inc()

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("There was a problem updating your linked git message: %s", err.Error()),
		},
	})
	if err != nil {
		log.WithError(err).Error("could not respond to user")
	}
}