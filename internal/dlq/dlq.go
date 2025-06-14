package dlq

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/dpe27/esk-puls/config"
	"github.com/dpe27/esk-puls/internal/job"
	"github.com/dpe27/esk-puls/pkg/log"
	"github.com/dpe27/esk-puls/pkg/utils"
	"github.com/redis/go-redis/v9"
)

const queueKey = "esk-puls:dlq"

type DLQJob struct {
	ID         string      `json:"id"`
	FailedJob  job.HttpJob `json:"failed_job"`
	Error      string      `json:"error"`
	RetryCount int         `json:"retry_count"`
	LastTried  time.Time   `json:"last_tried"`
}

type DeadLetterQueue interface {
	Push(ctx context.Context, job interface{}) error
	Pop(ctx context.Context, destination interface{}) error

	Ping(ctx context.Context) error
	Close(ctx context.Context)
}

type redisDLQ struct {
	logger *log.Logger
	client *redis.Client
}

func NewDeadLetterQueue(cfg *config.Config) DeadLetterQueue {
	logger := log.With("service", "dlq")
	client := redis.NewClient(&redis.Options{
		Addr:            cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password:        cfg.Redis.Password,
		ClientName:      cfg.Redis.ClientName,
		Username:        cfg.Redis.Username,
		MaxRetries:      cfg.Redis.MaxRetries,
		PoolSize:        cfg.Redis.PoolSize,
		MaxIdleConns:    cfg.Redis.MaxIdleConns,
		MaxActiveConns:  cfg.Redis.MaxActiveConns,
		ConnMaxIdleTime: time.Duration(cfg.Redis.MaxIdleConns) * time.Minute,
		ConnMaxLifetime: time.Duration(cfg.Redis.MaxLifeTime) * time.Minute,
	})

	return &redisDLQ{
		logger: logger,
		client: client,
	}
}

func (r *redisDLQ) Push(ctx context.Context, job interface{}) error {
	if job == nil {
		return errors.New("job cannot be nil")
	}

	data, err := json.Marshal(job)
	if err != nil {
		r.logger.Error(ctx, utils.ErrorMarshalJobBody, "error", err)
		return err
	}

	r.logger.Info(ctx, "Pushing job to DLQ", "job_id", job.(DLQJob).ID)
	if err := r.client.LPush(ctx, queueKey, data).Err(); err != nil {
		r.logger.Error(ctx, "Failed to push job to DLQ", "error", err)
		return err
	}
	return nil
}

func (r *redisDLQ) Pop(ctx context.Context, destination interface{}) error {
	if destination == nil {
		return errors.New("destination cannot be nil")
	}

	data, err := r.client.RPop(ctx, queueKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			r.logger.Info(ctx, "No jobs in DLQ to pop")
			return nil
		}
		r.logger.Error(ctx, "Failed to pop job from DLQ", "error", err)
		return err
	}

	if err := json.Unmarshal(data, destination); err != nil {
		r.logger.Error(ctx, utils.ErrorUnmarshalJobBody, "error", err)
		return err
	}

	r.logger.Info(ctx, "Popped job from DLQ", "job_id", destination.(DLQJob).ID)
	return nil
}

func (r *redisDLQ) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *redisDLQ) Close(ctx context.Context) {
	if err := r.client.Close(); err != nil {
		r.logger.Error(ctx, "Failed to close Redis connection", "error", err)
	} else {
		r.logger.Info(ctx, "Redis connection closed successfully")
	}
}
