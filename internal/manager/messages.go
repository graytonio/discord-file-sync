package manager

import (
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/graytonio/discord-git-sync/internal/db"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DiscordSessionInterface interface {
	ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageEditEmbed(channelID string, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageDelete(embedID string, messageID string, options ...discordgo.RequestOption) error
}

// Fetches url content, builds embed object based on guild settings, and sends new embed message
func CreateNewLinkedMessage(log *logrus.Entry, s DiscordSessionInterface, dbConn *gorm.DB, contentURL *url.URL, guildID string, channelID string) (string, error) {
	content, err := fetchPage(log, contentURL)
	if err != nil {
		log.WithError(err).Error("could not fetch url content")
		return "", err
	}

	pageBreakSetting, err := db.GetGuildSetting(dbConn, guildID, db.PageBreakEnabled)
	if err != nil {
		log.WithError(err).Error("could not fetch guild settings")
		return "", err
	}

	embeds, err := buildEmbedContents(content, contentURL, pageBreakSetting.Enabled)
	if err != nil {
		log.WithError(err).Error("could not build embed objects")
		return "", err
	}

	msgIDs := make([]string, len(embeds))
	for i, emb := range embeds {
		msg, err := s.ChannelMessageSendEmbed(channelID, emb)
		if err != nil {
			log.WithError(err).Error("could not send new embed")
			return "", err
		}

		msgIDs[i] = msg.ID
	}

	updateRateSetting, err := db.GetGuildSetting(dbConn, guildID, db.MessageAutoUpdateRate)
	if err != nil {
		log.WithError(err).Error("could not fetch guild settings")
		return "", err
	}

	err = dbConn.Create(&db.LinkedMessage{
		GuildID:      guildID,
		ChannelID:    channelID,
		MessageID:    msgIDs[0],
		MessageChain: msgIDs,
		LinkedPage:   datatypes.URL(*contentURL),
		NextUpdate:   time.Now().Add(updateRateSetting.DurationValue),
	}).Error
	if err != nil {
		log.WithError(err).Error("cloud not save message data in db")
		return "", err
	}

	return msgIDs[0], nil
}

// Fetches page configured for message, refreshes data, pushes updates to discord
func UpdateMessage(log *logrus.Entry, s DiscordSessionInterface, dbConn *gorm.DB, guildID string, channelID string, messageID string) error {
	linkedMessage := db.LinkedMessage{}
	err := dbConn.Where(&db.LinkedMessage{
		MessageID: messageID,
	}).First(&linkedMessage).Error
	if err != nil {
		log.WithError(err).Error("could not fetch data for linked message")
		return err
	}

	content, err := fetchPage(log, (*url.URL)(&linkedMessage.LinkedPage))
	if err != nil {
		log.WithError(err).Error("could not fetch url content")
		return err
	}

	pageBreakSetting, err := db.GetGuildSetting(dbConn, guildID, db.PageBreakEnabled)
	if err != nil {
		log.WithError(err).Error("could not fetch guild settings")
		return err
	}

	embeds, err := buildEmbedContents(content, (*url.URL)(&linkedMessage.LinkedPage), pageBreakSetting.Enabled)
	if err != nil {
		log.WithError(err).Error("could not build embed objects")
		return err
	}
	
	msgIds := make([]string, len(embeds))
	for i, e := range embeds {
		var msg *discordgo.Message
		if i >= len(linkedMessage.MessageChain) { // Handle more pages than previous message data
			msg, err = s.ChannelMessageSendEmbed(channelID, e)
		} else {
			msg, err = s.ChannelMessageEditEmbed(linkedMessage.ChannelID, linkedMessage.MessageChain[i], e)
		}

		if err != nil {
			if restErr, ok := err.(*discordgo.RESTError); ok {
				err = handleRESTErrors(log, s, dbConn, &linkedMessage, restErr)
				if err == nil {
					return nil
				}
			}
			
			log.WithError(err).Error("could not update message chain")
			return err
		}

		msgIds[i] = msg.ID
	}

	for _, m := range linkedMessage.MessageChain {
		if slices.Contains(msgIds, m) { // Do not delete any messages we updated with content above
			continue
		}

		err = s.ChannelMessageDelete(linkedMessage.ChannelID, m)
		if err != nil {
			if restErr, ok := err.(*discordgo.RESTError); ok && restErr.Message.Code == discordgo.ErrCodeUnknownMessage {
				log.Debug("skipping delete of already-deleted message")
				continue
			}
			log.WithError(err).Error("could not clean up message chain")
			return err
		}
	}

	updateRateSetting, err := db.GetGuildSetting(dbConn, guildID, db.MessageAutoUpdateRate)
	if err != nil {
		log.WithError(err).Error("could not fetch guild settings")
		return err
	}

	linkedMessage.NextUpdate = time.Now().Add(updateRateSetting.DurationValue)
	linkedMessage.MessageChain = msgIds
	linkedMessage.MessageID = msgIds[0]
	err = dbConn.Save(&linkedMessage).Error
	if err != nil {
		log.WithError(err).Error("could not save message chain data")
		return err
	}

	return nil
}

func handleRESTErrors(log *logrus.Entry, s DiscordSessionInterface, dbConn *gorm.DB, linkedMessage *db.LinkedMessage, err *discordgo.RESTError) error {
	log.Debug("handling rest error")
	if err.Message == nil {
		log.WithError(err).Warn("rest error with no message body")
		return err
	}
	switch err.Message.Code {
	case discordgo.ErrCodeUnknownMessage, discordgo.ErrCodeUnknownChannel:
		log.Debug("deleting stale message")
		return dbConn.Delete(linkedMessage).Error
	case discordgo.ErrCodePerformedOperationOnArchivedThread:
		// TODO(roadmap) Figure out what to do about archived threads
	default:
		log.WithError(err).Warn("unknown rest error")
		return err
	}
	return nil
}

// Fetch content of url as string
func fetchPage(log *logrus.Entry, link *url.URL) (string, error) {
	res, err := http.Get(link.String())
	if err != nil {
		log.WithError(err).Debug("could not fetch page")
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.WithError(err).Debug("could not parse body")
		return "", err
	}

	return string(body), nil
}

var pageBreakFinder = regexp.MustCompile(`(?m)^-{3,}$`) // TODO(maint) Remove need for regex

// Break raw page content into embed blocks to send or update
func buildEmbedContents(content string, link *url.URL, pageBreakSetting bool) ([]*discordgo.MessageEmbed, error) {
	// No further processing required return single embed object
	if !pageBreakSetting {
		return []*discordgo.MessageEmbed{{
			Description: content,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Source",
					Value: link.String(),
				},
			}}}, nil
	}

	pages := pageBreakFinder.Split(content, -1)
	embeds := []*discordgo.MessageEmbed{}

	for _, p := range pages {
		embeds = append(embeds, &discordgo.MessageEmbed{
			Description: p,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Source",
					Value: link.String(),
				},
			},
		})
	}

	return embeds, nil
}
