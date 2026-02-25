package service

import (
	"errors"
	"sync"
	"time"

	"github.com/shiroha/subdux/internal/model"
)

type notificationDispatchJob struct {
	subscriptionID  uint
	channel         model.NotificationChannel
	notifyDate      time.Time
	message         string
	targetEmail     string
	subscriptionURL string
}

func (s *NotificationService) dispatchNotificationChannel(channel model.NotificationChannel, targetEmail, message, subscriptionURL string) error {
	switch channel.Type {
	case "smtp":
		return s.sendSMTP(channel, targetEmail, message)
	case "resend":
		return s.sendResend(channel, message)
	case "telegram":
		return s.sendTelegram(channel, message)
	case "webhook":
		return s.sendWebhook(channel, message)
	case "gotify":
		return s.sendGotify(channel, message)
	case "ntfy":
		return s.sendNtfy(channel, message, subscriptionURL)
	case "bark":
		return s.sendBark(channel, message)
	case "serverchan":
		return s.sendServerChan(channel, message)
	case "feishu":
		return s.sendFeishu(channel, message)
	case "wecom":
		return s.sendWeCom(channel, message)
	case "dingtalk":
		return s.sendDingTalk(channel, message)
	case "pushdeer":
		return s.sendPushDeer(channel, message)
	case "pushplus":
		return s.sendPushplus(channel, message)
	case "pushover":
		return s.sendPushover(channel, message)
	case "napcat":
		return s.sendNapCat(channel, message)
	default:
		return errors.New("unsupported channel type")
	}
}

func (s *NotificationService) dispatchNotificationJobs(userID uint, dispatchJobs []notificationDispatchJob) error {
	workerCount := notificationDispatchWorkerCount(len(dispatchJobs))
	if workerCount == 0 {
		return nil
	}

	jobs := make(chan notificationDispatchJob, len(dispatchJobs))
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				sendErr := s.dispatchNotificationChannel(job.channel, job.targetEmail, job.message, job.subscriptionURL)

				logEntry := model.NotificationLog{
					UserID:         userID,
					SubscriptionID: job.subscriptionID,
					ChannelType:    job.channel.Type,
					NotifyDate:     job.notifyDate,
					SentAt:         time.Now().UTC(),
				}

				if sendErr != nil {
					logEntry.Status = "failed"
					logEntry.Error = sendErr.Error()
				} else {
					logEntry.Status = "sent"
				}

				s.DB.Create(&logEntry)
			}
		}()
	}

	for _, job := range dispatchJobs {
		jobs <- job
	}
	close(jobs)
	wg.Wait()

	return nil
}
