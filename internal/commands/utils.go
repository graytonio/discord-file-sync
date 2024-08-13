package commands

import (
	"io"
	"net/http"
	"net/url"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

func commandOptionsToMap(opts []*discordgo.ApplicationCommandInteractionDataOption) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	results := map[string]*discordgo.ApplicationCommandInteractionDataOption{}
	for _, v := range opts {
		results[v.Name] = v
	}
	return results
}

func fetchPage(log *logrus.Entry, link *url.URL) (string, error) {
	// Fetch content
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