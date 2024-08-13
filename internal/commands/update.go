package commands

import (
	"fmt"
	"net/url"

	"github.com/bwmarrin/discordgo"
	"github.com/graytonio/discord-git-sync/internal/db"
	"github.com/graytonio/discord-git-sync/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UpdateCommand struct{
	db *gorm.DB
}

var _ SlashCommand = &UpdateCommand{}

// GetDefinition implements SlashCommand.
func (u *UpdateCommand) GetDefinition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name: "update-message",
		Description: "Update a linked message with the latest content",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name: "message",
				Description: "Link to message to update",
				Required: true,
				Type: discordgo.ApplicationCommandOptionString,
			},
		},
	}
}

// GetHandler implements SlashCommand.
func (u *UpdateCommand) GetHandler() func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		log := logrus.WithField("user", i.Member.User.ID).WithField("guild", i.GuildID).WithField("interaction_id", i.Interaction.ID)
		log.Info("updating linked message")

		opts := commandOptionsToMap(i.ApplicationCommandData().Options)

		// TODO Preprocess message option to strip out message id

		linkedMessage := db.LinkedMessage{}
		err := u.db.Where(db.LinkedMessage{
			MessageID: opts["message"].StringValue(),
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

		msg, err := s.ChannelMessageEditEmbed(linkedMessage.ChannelID, linkedMessage.MessageID, &discordgo.MessageEmbed{
			Description: content,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name: "Source",
					Value: linkedMessage.LinkedPage.String(),
				},
			},
		})
		if err != nil {
			log.WithError(err).Error("could not update message content")
			u.sendErrorResponse(s, i, log, err)
			return
		}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: fmt.Sprintf("Message content updated: https://discord.com/channels/%s/%s/%s", linkedMessage.GuildID, linkedMessage.ChannelID, msg.ID),
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