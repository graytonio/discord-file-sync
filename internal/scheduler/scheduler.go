package scheduler

import (
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/graytonio/discord-git-sync/internal/db"
	"github.com/graytonio/discord-git-sync/internal/manager"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// When message is linked/updated create column that stores next update time. Update time is based on guild settings and billing allowance
// On scheduled interval look for messages past their next update time
// Update all messages as needed (Open to refactor to queue system at later date)

func InitJobScheduler(db *gorm.DB, s manager.DiscordSessionInterface) (gocron.Scheduler, error) {
	var err error
	// locker, err := gormlock.NewGormLocker(db, os.Getenv("WORKER_ID")) // TODO Update dependency after bump
	// if err != nil {
	//   return err
	// }
	
	scheduler, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))
	if err != nil {
	  return nil, err
	}

	scheduler.NewJob(
		gocron.CronJob("*/5 * * * *", false), // TODO Configurable
		gocron.NewTask(fetchAndDispatchMessageUpdates, db, scheduler, s),
	)
	scheduler.Start()

	return scheduler, nil
}

func fetchAndDispatchMessageUpdates(dbConn *gorm.DB, scheduler gocron.Scheduler, s manager.DiscordSessionInterface) {
	logrus.Info("fetching out of date messages")
	messages, err := db.GetMessagesToUpdate(dbConn)
	if err != nil {
	  logrus.WithError(err).Error("could not fetch messages for auto update")
	  return
	}

	for _, m := range messages {
		log := logrus.WithFields(logrus.Fields{
			"command": "autoupdate",
			"guild": m.GuildID,
			"message_id": m.MessageID,
		})

		log.Info("auto updating linked message")

		scheduler.NewJob(
			gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
			gocron.NewTask(manager.UpdateMessage, log, s, dbConn, m.GuildID, m.ChannelID, m.MessageID),
		)
	}
}