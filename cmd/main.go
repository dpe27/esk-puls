package main

import (
	"context"
	"time"

	"github.com/dpe27/esk-puls/config"
	"github.com/dpe27/esk-puls/internal/dlq"
	"github.com/dpe27/esk-puls/internal/httpclient"
	"github.com/dpe27/esk-puls/internal/job"
	"github.com/dpe27/esk-puls/pkg/log"
	cronlogger "github.com/dpe27/esk-puls/pkg/log/cron"

	"github.com/robfig/cron/v3"
)

func main() {
	ctx := context.Background()
	cfg := config.NewConfig()

	loc, err := time.LoadLocation(cfg.App.Location)
	if err != nil {
		log.Error(ctx, "Failed to load location", "error", err)
		return
	}

	jobs, err := job.LoadJobsFromDir()
	if err != nil {
		log.Error(ctx, "Failed to load jobs", "error", err)
		return
	}

	queue := dlq.NewDeadLetterQueue(cfg)
	if err := queue.Ping(ctx); err != nil {
		log.Error(ctx, "Failed to connect to DLQ", "error", err)
		return
	}

	cliOpt := httpclient.ClientOptBuilder().
		ServiceName("esk-puls").
		Build()
	httpCli := httpclient.NewHttpClient(cliOpt)
	dlqWorker := dlq.NewDLQWorker(queue, httpCli)

	workerCtx, stopWorker := context.WithCancel(ctx)
	dlqWorker.Start(workerCtx)
	defer stopWorker()

	c := cron.New(
		cron.WithLocation(loc),
		cron.WithLogger(cronlogger.NewCronLogger()),
	)
	scheduleJobs(ctx, c, jobs, httpCli, queue, loc)
	c.Start()
	defer c.Stop()
}

func scheduleJobs(
	ctx context.Context,
	c *cron.Cron,
	jobs []job.HttpJob,
	client httpclient.HttpClient,
	queue dlq.DeadLetterQueue,
	loc *time.Location,
) {
	for _, j := range jobs {
		jobRunner := job.NewJobRunner(&j, client, loc)
		_, err := c.AddFunc(j.Schedule, func() {
			log.Info(ctx, "Executing scheduled job", "name", j.Name, "schedule", j.Schedule)
			if lastTried, err := jobRunner.Run(); err != nil {
				log.Error(ctx, "Job execution failed", "name", j.Name, "error", err)
				dlqJob := dlq.DLQJob{
					ID:         j.Name,
					FailedJob:  j,
					Error:      err.Error(),
					RetryCount: j.MaxRetries,
					LastTried:  lastTried,
				}
				if err := queue.Push(ctx, dlqJob); err != nil {
					log.Error(ctx, "Failed to push job to DLQ", "error", err)
				}
			}
		})

		if err != nil {
			log.Error(ctx, "Failed to schedule job", "name", j.Name, "error", err)
		}
	}
}
