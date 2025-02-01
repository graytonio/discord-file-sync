package scheduler

import (
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/graytonio/discord-git-sync/internal/db"
	"github.com/graytonio/discord-git-sync/internal/manager"
	"github.com/graytonio/discord-git-sync/internal/metrics"
	"github.com/graytonio/discord-git-sync/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// When message is linked/updated create column that stores next update time. Update time is based on guild settings and billing allowance
// On scheduled interval look for messages past their next update time
// Update all messages as needed (Open to refactor to queue system at later date)

func InitJobScheduler(db *gorm.DB, s manager.DiscordSessionInterface) (gocron.Scheduler, error) {
	var err error
	// locker, err := gormlock.NewGormLocker(db, os.Getenv("WORKER_ID")) // TODO(roadmap) Update dependency after bump
	// if err != nil {
	//   return err
	// }
	
	scheduler, err := gocron.NewScheduler(
		gocron.WithLocation(time.UTC),
		// TODO(maint) Add prometheus monitor
	)
	if err != nil {
	  return nil, err
	}

	scheduler.NewJob(
		gocron.CronJob(utils.GetEnv("UPDATE_TASK_CRON", "*/5 * * * *"), false),
		gocron.NewTask(fetchAndDispatchMessageUpdates, db, scheduler, s),
		gocron.WithName("message-batch-collection"),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
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

		log.Debug("auto updating linked message")
		scheduler.NewJob(
			gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
			gocron.NewTask(manager.UpdateMessage, log, s, dbConn, m.GuildID, m.ChannelID, m.MessageID),
			gocron.WithName(fmt.Sprintf("message-update-%s", m.MessageID)),
			gocron.WithEventListeners(
				gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
					metrics.CommandsServed.With(prometheus.Labels{"command": "auto-update"}).Inc()
				}),
				gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) {
					metrics.CommandsFailed.With(prometheus.Labels{"command": "auto-update"}).Inc()
				}),
			),
		)
	}
}