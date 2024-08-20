package commands

import (
	"net/url"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestBuildEmbedContents(t *testing.T) {
	// Define test cases
	tests := []struct {
		name             string
		content          string
		link             *url.URL
		pageBreakSetting bool
		expectedEmbeds   []*discordgo.MessageEmbed
		expectError      bool
	}{
		{
			name:    "No page break, setting off",
			content: "This is a single page content.",
			link: &url.URL{
				Scheme: "https",
				Host:   "example.com",
				Path:   "/path",
			},
			pageBreakSetting: false,
			expectedEmbeds: []*discordgo.MessageEmbed{
				{
					Description: "This is a single page content.",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Source",
							Value: "https://example.com/path",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:    "Single page break, setting on",
			content: "This is page one.\n---\nThis is page two.",
			link: &url.URL{
				Scheme: "https",
				Host:   "example.com",
				Path:   "/path",
			},
			pageBreakSetting: true,
			expectedEmbeds: []*discordgo.MessageEmbed{
				{
					Description: "This is page one.\n",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Source",
							Value: "https://example.com/path",
						},
					},
				},
				{
					Description: "\nThis is page two.",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Source",
							Value: "https://example.com/path",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:    "Multiple page breaks, setting on",
			content: "Page 1\n---\nPage 2\n---\nPage 3",
			link: &url.URL{
				Scheme: "https",
				Host:   "example.com",
				Path:   "/path",
			},
			pageBreakSetting: true,
			expectedEmbeds: []*discordgo.MessageEmbed{
				{
					Description: "Page 1\n",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Source",
							Value: "https://example.com/path",
						},
					},
				},
				{
					Description: "\nPage 2\n",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Source",
							Value: "https://example.com/path",
						},
					},
				},
				{
					Description: "\nPage 3",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Source",
							Value: "https://example.com/path",
						},
					},
				},
			},
			expectError: false,
		},
	}

	// Execute test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embeds, err := buildEmbedContents(tt.content, tt.link, tt.pageBreakSetting)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedEmbeds, embeds)
			}
		})
	}
}

func TestStripDiscordLink(t *testing.T) {
	tests := []struct{
		name string
		link string
		expectedMessageID string
	}{
		{
			name: "discord link",
			link: "https://discord.com/channels/379815890276843521/1271348383877042186/1275288085705261130",
			expectedMessageID: "1275288085705261130",
		},
		{
			name: "just id",
			link: "1275288085705261130",
			expectedMessageID: "1275288085705261130",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := stripDiscordLinkMessageID(tt.link)
			assert.Equal(t, tt.expectedMessageID, out)
		})
	}
}