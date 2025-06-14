package dlq

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/dpe27/esk-puls/internal/httpclient"
	"github.com/dpe27/esk-puls/pkg/log"
	"github.com/dpe27/esk-puls/pkg/utils"
)

const DeadLetterPath = "/dead-letter"

type DLQWorker struct {
	queue  DeadLetterQueue
	cli    httpclient.HttpClient
	logger *log.Logger
}

func NewDLQWorker(queue DeadLetterQueue, client httpclient.HttpClient) *DLQWorker {
	return &DLQWorker{
		queue:  queue,
		cli:    client,
		logger: log.With("service", "dlq_worker"),
	}
}
func (w *DLQWorker) Start(ctx context.Context) {
	w.logger.Info(ctx, "Starting DLQ worker")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info(ctx, "Stopping DLQ worker")
			w.queue.Close(ctx)
			return

		default:
			var job DLQJob
			if err := w.queue.Pop(ctx, &job); err != nil {
				w.logger.Error(ctx, "Failed to pop job from DLQ", "error", err)
				continue
			}
			w.logger.Info(ctx, "Processing job from DLQ", "job_id", job.ID)
			reqBody, err := json.Marshal(job)
			if err != nil {
				w.logger.Error(ctx, utils.ErrorMarshalJobBody, "error", err)
				continue
			}

			req, err := http.NewRequestWithContext(
				ctx,
				http.MethodPost,
				DeadLetterPath,
				strings.NewReader(string(reqBody)),
			)
			if err != nil {
				w.logger.Error(ctx, utils.ErrorCreateRequest, "error", err)
				continue
			}

			opts := httpclient.ReqOptBuilder().
				Log().LogReqBodyOnlyError().
				LogResBody().
				LoggedResBody([]string{}).
				LoggedReqBody([]string{}).
				Build()

			func() {
				resp, err := w.cli.Do(req, opts)
				if err != nil {
					w.logger.Error(ctx, "Failed to execute job request", "error", err)
					return
				}
				defer func() {
					if err := resp.Body.Close(); err != nil {
						w.logger.Error(ctx, utils.ErrorCloseResponseBody, "error", err)
					}
				}()

				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					respBody, err := io.ReadAll(resp.Body)
					if err != nil {
						w.logger.Error(ctx, utils.ErrorReadBody, "error", err)
						return
					}
					w.logger.Error(ctx, "failed to process dead letter job", "name", job.FailedJob.Name, "status_code", resp.StatusCode, "response_body", string(respBody))
					return
				}

				w.logger.Info(ctx, "Successfully processed job from DLQ", "job_id", job.ID, "name", job.FailedJob.Name)
			}()
		}
	}
}
