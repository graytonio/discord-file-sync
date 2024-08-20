package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/graytonio/discord-git-sync/internal/db"
	"github.com/graytonio/discord-git-sync/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingsCommand struct {
	db *gorm.DB
}

var _ SlashCommand = &SettingsCommand{}

// GetDefinition implements SlashCommand.
func (sc *SettingsCommand) GetDefinition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "update-setting",
		Description: "Update a guild setting",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "option",
				Description: "Option to update",
				Required:    true,
				Type:        discordgo.ApplicationCommandOptionString,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Page Break Embeds",
						Value: db.PageBreakEnabled,
					},
				},
			},
			{
				Name:        "setting",
				Description: "What to set the option to",
				Required:    true,
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
		},
	}
}

// GetHandler implements SlashCommand.
func (sc *SettingsCommand) GetHandler() func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		log := logrus.WithFields(logrus.Fields{
			"command":        "settings",
			"user":           i.Member.User.ID,
			"guild":          i.GuildID,
			"interaction_id": i.ID,
		})
		log.Info("updating user setting")

		opts := commandOptionsToMap(i.ApplicationCommandData().Options)

		err := sc.db.Model(&db.GuildSetting{}).
			Clauses(clause.OnConflict{
				UpdateAll: true,
			}).
			Create(&db.GuildSetting{
				GuildID: i.GuildID, 
				Setting: db.Setting(opts["option"].StringValue()), 
				Enabled: opts["setting"].BoolValue(),
			}).
			Error
		if err != nil {
		  log.WithError(err).Error("could not update db")
		  sc.sendErrorResponse(s, i, log, err)
		  return
		}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: fmt.Sprintf("Guild setting %s set to %t", opts["option"].StringValue(), opts["setting"].BoolValue()),
			},
		})
		if err != nil {
			metrics.CommandsFailed.With(prometheus.Labels{"command": "settings"}).Inc()
			log.WithError(err).Error("could not respond to user")
			return
		}
		metrics.CommandsServed.With(prometheus.Labels{"command": "settings"}).Inc()
	}

}

func (sc *SettingsCommand) sendErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, log *logrus.Entry, err error) {
	metrics.CommandsFailed.With(prometheus.Labels{"command": "settings"}).Inc()

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
