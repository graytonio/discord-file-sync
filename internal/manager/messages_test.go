package manager

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/graytonio/discord-git-sync/internal/db"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// MockDiscordSessionInterface implements the DiscordSessionInterface
// and is used to mock the Discord session.
type MockDiscordSessionInterface struct {}

func (m *MockDiscordSessionInterface) ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	return &discordgo.Message{ID: "mockMessageID"}, nil
}

func (m *MockDiscordSessionInterface) ChannelMessageEditEmbed(channelID string, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	return &discordgo.Message{ID: messageID}, nil
}

func (m *MockDiscordSessionInterface) ChannelMessageDelete(channelID string, messageID string, options ...discordgo.RequestOption) error {
	return nil
}

func setupMySQLContainer(t *testing.T) (*gorm.DB, func()) {
	req := testcontainers.ContainerRequest{
		Image:        "mysql:8",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "testpassword",
			"MYSQL_DATABASE":     "testdb",
			"MYSQL_USER":         "testuser",
			"MYSQL_PASSWORD":     "testpassword",
		},
		WaitingFor: wait.ForListeningPort("3306/tcp"),
	}

	mysqlContainer, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoError(t, err, "failed to start MySQL container")

	host, err := mysqlContainer.Host(context.Background())
	assert.NoError(t, err, "failed to get MySQL container host")

	port, err := mysqlContainer.MappedPort(context.Background(), "3306")
	assert.NoError(t, err, "failed to get MySQL container port")

	dsn := "testuser:testpassword@tcp(%s:%s)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	dbConn, err := db.InitDB(fmt.Sprintf(dsn, host, port.Port()))
	assert.NoError(t, err, "failed to connect to MySQL container")

	cleanup := func() {
		_ = mysqlContainer.Terminate(context.Background())
	}

	return dbConn, cleanup
}

func TestCreateNewLinkedMessage(t *testing.T) {
	log := logrus.NewEntry(logrus.New())

	// Setup MySQL test container
	dbConn, cleanup := setupMySQLContainer(t)
	defer cleanup()

	err := dbConn.Create(&db.GuildSetting{
		GuildID: "guildID",
		Setting: db.PageBreakEnabled,
		Enabled: true,
	}).Error
	assert.NoError(t, err, "could not seed db data")

	// Create a mock HTTP server to simulate fetching markdown content
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# Example Markdown Content"))
	}))
	defer httpServer.Close()

	contentURL, _ := url.Parse(httpServer.URL)

	// Mock Discord session interface
	discordSession := &MockDiscordSessionInterface{}

	// Run the function
	messageID, err := CreateNewLinkedMessage(log, discordSession, dbConn, contentURL, "guildID", "channelID")

	// Assertions
	assert.NoError(t, err, "error should be nil")
	assert.Equal(t, "mockMessageID", messageID, "message ID should match")

	// Verify database record
	var linkedMessage db.LinkedMessage
	result := dbConn.First(&linkedMessage, "guild_id = ? AND channel_id = ?", "guildID", "channelID")
	assert.NoError(t, result.Error, "linked message should exist in the database")
	assert.Equal(t, "mockMessageID", linkedMessage.MessageID, "message ID in the database should match")
	assert.Equal(t, *contentURL, (url.URL)(linkedMessage.LinkedPage), "linked page URL should match")
}

func TestUpdateMessage(t *testing.T) {
	log := logrus.NewEntry(logrus.New())

	// Setup MySQL test container
	dbConn, cleanup := setupMySQLContainer(t)
	defer cleanup()

	err := dbConn.Create(&db.GuildSetting{
		GuildID: "guildID",
		Setting: db.PageBreakEnabled,
		Enabled: true,
	}).Error
	assert.NoError(t, err, "could not seed db data")

	linkedPageURL, _ := url.Parse("https://example.com/page")
	err = dbConn.Create(&db.LinkedMessage{
		GuildID:      "guildID",
		ChannelID:    "channelID",
		MessageID:    "mockMessageID",
		MessageChain: db.MessageChain{"mockMessageID"},
		LinkedPage:   datatypes.URL(*linkedPageURL),
	}).Error
	assert.NoError(t, err, "could not seed db data")

	// Create a mock HTTP server to simulate fetching markdown content
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# Updated Markdown Content"))
	}))
	defer httpServer.Close()

	// Mock Discord session interface
	discordSession := &MockDiscordSessionInterface{}

	// Run the function
	err = UpdateMessage(log, discordSession, dbConn, "guildID", "channelID", "mockMessageID")

	// Assertions
	assert.NoError(t, err, "error should be nil")

	// Verify database record
	var linkedMessage db.LinkedMessage
	result := dbConn.First(&linkedMessage, "message_id = ?", "mockMessageID")
	assert.NoError(t, result.Error, "linked message should exist in the database")
	assert.Equal(t, "mockMessageID", linkedMessage.MessageID, "message ID in the database should match")
	assert.Equal(t, "mockMessageID", linkedMessage.MessageChain[0], "embed should be updated in the message chain")
	updatedURL, _ := url.Parse("https://example.com/page")
	assert.Equal(t, *updatedURL, (url.URL)(linkedMessage.LinkedPage), "linked page URL should match")
}

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