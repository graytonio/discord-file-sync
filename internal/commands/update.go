package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/graytonio/discord-git-sync/internal/manager"
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
		

		opts := commandOptionsToMap(i.ApplicationCommandData().Options)

		messageID := stripDiscordLinkMessageID(opts["message"].StringValue())

		log := logrus.WithFields(logrus.Fields{
			"command":        "update",
			"user":           i.Member.User.ID,
			"guild":          i.GuildID,
			"interaction_id": i.ID,
			"message_id": messageID,
		})
		log.Info("updating linked message")

		err := manager.UpdateMessage(log, s, u.db, i.GuildID, i.ChannelID, messageID)
		if err != nil {
			metrics.CommandsFailed.With(prometheus.Labels{"command": "update"}).Inc()
			log.WithError(err).Error("cloud not update message")
			u.sendErrorResponse(s, i, log, err)
			return
		}
		
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: fmt.Sprintf("Message content updated: https://discord.com/channels/%s/%s/%s", i.GuildID, i.ChannelID, messageID),
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