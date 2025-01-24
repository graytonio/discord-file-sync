package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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