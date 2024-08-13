package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/graytonio/discord-git-sync/internal/bot"
	"github.com/graytonio/discord-git-sync/internal/db"
	"github.com/graytonio/discord-git-sync/internal/metrics"
	"github.com/sirupsen/logrus"
)

func main() {
	dbConn, err := db.InitDB(os.Getenv("MYSQL_DB_DSN"))
	if err != nil {
	  logrus.WithError(err).Fatal("could not connect to db")
	}

	s, err := bot.InitBot(os.Getenv("DISCORD_BOT_TOKEN"), dbConn, os.Getenv("TEST_GUILD_ID"))
	if err != nil {
		logrus.WithError(err).Fatal("could not start bot")
	}
	defer s.Close()

	go metrics.StartMetrics()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	logrus.Info("Bot started")
	<-stop

	logrus.Info("Gracefully shutting down")
}
