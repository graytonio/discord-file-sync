package commands

import (
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

// Parse array of discord options into map
func commandOptionsToMap(opts []*discordgo.ApplicationCommandInteractionDataOption) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	results := map[string]*discordgo.ApplicationCommandInteractionDataOption{}
	for _, v := range opts {
		results[v.Name] = v
	}
	return results
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

var pageBreakFinder = regexp.MustCompile(`(?m)^-{3,}$`)
// Break raw page content into embed blocks to send or update
func buildEmbedContents(content string, link *url.URL, pageBreakSetting bool) ([]*discordgo.MessageEmbed, error) {
	// No further processing required return single embed object
	if !pageBreakSetting {
		return []*discordgo.MessageEmbed{
			{
				Description: content,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Source",
						Value: link.String(),
					},
				},
			},
		}, nil
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

func stripDiscordLinkMessageID(link string) string {
	parts := strings.Split(link, "/")
	return parts[len(parts) - 1]
}