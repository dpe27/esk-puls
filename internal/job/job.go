package job

import "time"

type HttpJob struct {
	Name                 string                 `json:"name" yaml:"name"`
	Url                  string                 `json:"url" yaml:"url"`
	Schedule             string                 `json:"schedule" yaml:"schedule"`
	Method               string                 `json:"method" yaml:"method"`
	Headers              map[string]string      `json:"headers" yaml:"headers"`
	Body                 map[string]interface{} `json:"body" yaml:"body"`
	MaxRetries           int                    `json:"max_retries" yaml:"max_retries"`
	BackoffIsExponential bool                   `json:"backoff_is_exponential" yaml:"backoff_is_exponential"`
	BackoffDelay         time.Duration          `json:"backoff_delay" yaml:"backoff_delay"`
}
