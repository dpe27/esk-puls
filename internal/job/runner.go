package job

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dpe27/esk-puls/internal/httpclient"
	"github.com/dpe27/esk-puls/pkg/log"
	"github.com/dpe27/esk-puls/pkg/utils"
)

const ErrorMarshalJobBody = "failed to marshal job body"

type JobRunner struct {
	cli    httpclient.HttpClient
	logger *log.Logger
	job    *HttpJob
	loc    *time.Location
}

func NewJobRunner(job *HttpJob, client httpclient.HttpClient, loc *time.Location) *JobRunner {
	return &JobRunner{
		logger: log.With("job", job.Name),
		job:    job,
		cli:    client,
		loc:    loc,
	}
}

func (r *JobRunner) Run() (time.Time, error) {
	ctx := context.Background()
	r.logger.Info(ctx, "Starting job", "name", r.job.Name, "url", r.job.Url, "method", r.job.Method)

	var (
		err       error
		lastTried time.Time
	)
	for i := 0; i <= r.job.MaxRetries; i++ {
		lastTried = time.Now().In(r.loc)

		err = r.execute(ctx)
		if err == nil {
			return lastTried, nil
		}
		r.logger.Error(ctx, "Job execution failed", "name", r.job.Name, "attempt", i+1, "error", err, "last_tried", lastTried)
	}

	r.logger.Error(ctx, "Job failed after maximum retries", "name", r.job.Name, "max_retries", r.job.MaxRetries)
	return lastTried, fmt.Errorf("job %q failed after %d attempts: %w", r.job.Name, r.job.MaxRetries+1, err)
}

func (r *JobRunner) execute(ctx context.Context) error {
	r.logger.Info(ctx, "Running job", "name", r.job.Name, "url", r.job.Url, "method", r.job.Method)

	reqBody, err := json.Marshal(r.job.Body)
	if err != nil {
		r.logger.Error(ctx, ErrorMarshalJobBody, "error", err)
		return err
	}

	req, err := http.NewRequestWithContext(ctx, r.job.Method, r.job.Url, strings.NewReader(string(reqBody)))
	if err != nil {
		r.logger.Error(ctx, utils.ErrorCreateRequest, "error", err)
		return err
	}
	for k, v := range r.job.Headers {
		req.Header.Set(k, v)
	}

	opts := httpclient.ReqOptBuilder().
		Log().LogReqBodyOnlyError().
		LogResBody().
		LoggedResBody([]string{}).
		LoggedReqBody([]string{}).
		Build()

	resp, err := r.cli.Do(req, opts)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			r.logger.Error(ctx, utils.ErrorCloseResponseBody, "error", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			r.logger.Error(ctx, utils.ErrorReadBody, "error", err)
		}
		return fmt.Errorf("job %q failed with status code %d: %s", r.job.Name, resp.StatusCode, string(respBody))
	}

	r.logger.Info(ctx, "Job completed successfully", "name", r.job.Name, "status_code", resp.StatusCode)
	return nil
}
