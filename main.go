package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/graytonio/discord-git-sync/internal/bot"
	"github.com/graytonio/discord-git-sync/internal/db"
	"github.com/graytonio/discord-git-sync/internal/metrics"
	"github.com/graytonio/discord-git-sync/internal/scheduler"
	"github.com/graytonio/discord-git-sync/internal/utils"
	"github.com/sirupsen/logrus"
)

func main() {
	logLevel, err := logrus.ParseLevel(utils.GetEnv("LOG_LEVEL", "INFO"))
	if err != nil {
	  logrus.WithField("LOG_LEVEL", utils.GetEnv("LOG_LEVEL", "INFO")).Warn("invalid log level defaulting to info")
	  logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)

	dbConn, err := db.InitDB(os.Getenv("MYSQL_DB_DSN"))
	if err != nil {
	  logrus.WithError(err).Fatal("could not connect to db")
	}

	s, err := bot.InitBot(os.Getenv("DISCORD_BOT_TOKEN"), dbConn, os.Getenv("TEST_GUILD_ID"))
	if err != nil {
		logrus.WithError(err).Fatal("could not start bot")
	}
	defer s.Close()

	scheduler, err := scheduler.InitJobScheduler(dbConn, s)
	if err != nil {
	  logrus.WithError(err).Fatal("could not init cron tasks")
	}
	defer scheduler.Shutdown()

	go metrics.StartMetrics()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	logrus.Info("Bot started")
	<-stop

	logrus.Info("Gracefully shutting down")
}