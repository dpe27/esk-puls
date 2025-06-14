package cronlogger

import (
	"context"

	"github.com/dpe27/esk-puls/pkg/log"
)

type cronlogger struct {
	logger *log.Logger
}

func NewCronLogger() *cronlogger {
	return &cronlogger{
		logger: log.With("service", "cron"),
	}
}

func (c *cronlogger) Info(msg string, keysAndValues ...interface{}) {
	c.logger.Info(context.Background(), msg, keysAndValues...)
}

func (c *cronlogger) Error(err error, msg string, keysAndValues ...interface{}) {
	c.logger.Error(context.Background(), msg, append(keysAndValues, "error", err)...)
}
