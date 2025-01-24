package commands

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Parse array of discord options into map
func commandOptionsToMap(opts []*discordgo.ApplicationCommandInteractionDataOption) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	results := map[string]*discordgo.ApplicationCommandInteractionDataOption{}
	for _, v := range opts {
		results[v.Name] = v
	}
	return results
}

func stripDiscordLinkMessageID(link string) string {
	parts := strings.Split(link, "/")
	return parts[len(parts) - 1]
}