package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/dpe27/esk-puls/pkg/log"
	"github.com/dpe27/esk-puls/pkg/utils"
)

const (
	invalidStatusCode = 0
	bodyBytesLimit    = 1024
	mask              = "********"

	requestBodyLogKey  = "body_request"
	responseBodyLogKey = "body_response"
	urlLogKey          = "url"
	durationLogKey     = "duation"
	methodLogKey       = "method"
	protocolLogKey     = "protocol"
	serviceLogKey      = "service"
	statusCodeLogKey   = "status_code"

	unmatchedTypeMsg = "unmatched type"
	keyNotFoundMsg   = "key not found"
	emptyBodyMsg     = " is empty"
)

type (
	HttpClient interface {
		Do(req *http.Request, opts *reqOpt) (*http.Response, error)
	}

	httpClient struct {
		client *http.Client
		logger *log.Logger
	}
)

func NewHttpClient(args *clientOpt) HttpClient {
	logger := log.With("service", args.serviceName)
	if args.client != nil {
		return &httpClient{
			client: args.client,
			logger: logger,
		}
	}

	defaultTransport := http.DefaultTransport.(*http.Transport)

	transport := defaultTransport.Clone()
	transport.MaxConnsPerHost = args.maxIdleConnsPerHost
	transport.ResponseHeaderTimeout = args.responseHeaderTimeout

	return &httpClient{
		logger: logger,
		client: &http.Client{
			Timeout:   args.timeout,
			Transport: transport,
		},
	}
}

func (h *httpClient) Do(req *http.Request, opts *reqOpt) (*http.Response, error) {
	fields := make(map[string]interface{})
	args := h.preReq(req, opts, fields)

	var (
		res     *http.Response
		err     error
		start   time.Time
		retries = 0
	)
	for h.shouldRetry(err, res) && retries <= int(args.retryTimes) {
		time.Sleep(h.backoff(retries))
		start = time.Now()
		res, err = h.client.Do(req)
		if args.canLog {
			h.postReq(req, res, err, start, args, fields)
		}
		retries++
	}

	return res, err
}

func (h *httpClient) preReq(
	req *http.Request,
	opts *reqOpt,
	fields map[string]interface{},
) *reqOpt {
	u := h.maskUrl(req.URL.String(), opts.markedQueryParamKeys)
	if u == "" {
		fields[urlLogKey] = utils.ErrorParseUrl
	}
	fields[urlLogKey] = u
	req.Body = h.logBody(
		req.Body,
		requestBodyLogKey,
		opts.loggedRequestBody,
		true,
		fields,
	)
	return opts
}

func (h *httpClient) postReq(
	req *http.Request,
	res *http.Response,
	err error,
	start time.Time,
	args *reqOpt,
	fields map[string]interface{},
) {
	fields[durationLogKey] = time.Since(start)
	fields[methodLogKey] = req.Method
	fields[protocolLogKey] = req.Proto

	hasErr := (err != nil || res.StatusCode/100 == 4 || res.StatusCode/100 == 5)
	if !(hasErr && args.canLogRequestBodyOnlyError) && !args.canLogRequestBody {
		delete(fields, requestBodyLogKey)
	}
	if err != nil {
		h.outputLog(req.Context(), invalidStatusCode, err, fields)
		return
	}

	fields[statusCodeLogKey] = res.StatusCode
	if args.canLogResponseBody || args.canLogResponseBodyOnlyError {
		res.Body = h.logBody(
			res.Body,
			responseBodyLogKey,
			args.loggedRequestBody,
			false,
			fields,
		)
	}
	if !(hasErr && args.canLogResponseBodyOnlyError) && !args.canLogResponseBody {
		delete(fields, responseBodyLogKey)
	}

	h.outputLog(req.Context(), res.StatusCode, err, fields)
}

func (h *httpClient) logBody(
	b io.Reader,
	logKey string,
	loggedKeys []string,
	canLimit bool,
	fields map[string]interface{},
) io.ReadCloser {
	if b == nil {
		fields[logKey] = logKey + emptyBodyMsg
		return nil
	}

	if len(loggedKeys) == 0 && loggedKeys != nil {
		return h.logFilteredBody(b, logKey, loggedKeys, fields)
	}

	buf, err := io.ReadAll(b)
	if err != nil {
		fields[logKey] = utils.ErrorReadBody + logKey
		return io.NopCloser(bytes.NewBuffer(buf))
	}

	logBody := string(buf)
	if canLimit && len(buf) > bodyBytesLimit {
		logBody = string(buf[:bodyBytesLimit])
	}
	fields[logKey] = logBody
	return io.NopCloser(bytes.NewBuffer(buf))
}

func (h *httpClient) logFilteredBody(
	b io.Reader,
	logKey string,
	loggedKeys []string,
	fields map[string]interface{},
) io.ReadCloser {
	var bodyBuffer bytes.Buffer
	var result interface{}

	body := io.NopCloser(io.TeeReader(b, &bodyBuffer))
	err := json.NewDecoder(body).Decode(&result)
	if err != nil {
		fields[logKey] = utils.ErrorDecodeBody
	}

	switch result.(type) {
	case map[string]interface{}:
		fields[logKey] = h.filterJsonStruct(result, loggedKeys)
	case []interface{}:
		fields[logKey] = h.filterJsonArray(result, loggedKeys)
	default:
		fields[logKey] = unmatchedTypeMsg
	}

	return io.NopCloser(&bodyBuffer)
}

func (h *httpClient) filterJsonStruct(
	result interface{}, keys []string,
) map[string]interface{} {
	loggedResult := make(map[string]interface{})
	for _, key := range keys {
		loggedResult[key] = result.(map[string]interface{})[key]
	}
	return loggedResult
}

func (h *httpClient) filterJsonArray(
	result interface{}, keys []string,
) []map[string]interface{} {
	loggedResult := make([]map[string]interface{}, 0)
	for _, r := range result.([]interface{}) {
		loggedResult = append(loggedResult, make(map[string]interface{}))
		for _, key := range keys {
			v, ok := r.(map[string]interface{})[key]
			if ok {
				loggedResult[len(loggedResult)-1][key] = v
				continue
			}
		}
	}
	return loggedResult
}

// maskUrl func masks specified query parameters in the URL by
// replacing their values with a constant mask.
func (h *httpClient) maskUrl(u string, maskedKeys []string) string {
	if len(maskedKeys) == 0 {
		return u
	}

	pu, err := url.Parse(u)
	if err != nil {
		return u
	}

	p := pu.Query()
	for _, k := range maskedKeys {
		p.Set(k, mask)
	}
	pu.RawQuery = p.Encode()
	return pu.String()
}

func (h *httpClient) shouldRetry(err error, res *http.Response) bool {
	if res == nil ||
		res.StatusCode == http.StatusBadGateway ||
		res.StatusCode == http.StatusServiceUnavailable ||
		res.StatusCode == http.StatusRequestTimeout ||
		res.StatusCode == http.StatusGatewayTimeout ||
		os.IsTimeout(err) {

		return true
	}

	return false
}

func (h *httpClient) backoff(retries int) time.Duration {
	return time.Duration(math.Pow(2, float64(retries))) * time.Second
}

func (h *httpClient) outputLog(ctx context.Context, statusCode int, err error, fields map[string]interface{}) {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}

	if err != nil || statusCode/100 == 5 {
		h.logger.With(args...).Error(ctx, err.Error())
		return
	}

	msg := "Request completed with HttpClient"
	h.logger.With(args...).Info(ctx, msg)
}
