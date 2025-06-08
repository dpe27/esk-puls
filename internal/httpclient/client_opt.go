package httpclient

import (
	"net/http"
	"time"
)

type (
	clientOptBuilder struct {
		setters []func(*clientOpt)
	}

	clientOpt struct {
		client                *http.Client
		maxIdleConnsPerHost   int
		timeout               time.Duration
		responseHeaderTimeout time.Duration
		serviceName           string
	}
)

func ClientOptBuilder() *clientOptBuilder {
	return &clientOptBuilder{}
}

func (b *clientOptBuilder) Client(c *http.Client) *clientOptBuilder {
	b.setters = append(b.setters, func(co *clientOpt) {
		co.client = c
	})
	return b
}

func (b *clientOptBuilder) Timeout(timeout time.Duration) *clientOptBuilder {
	b.setters = append(b.setters, func(co *clientOpt) {
		co.timeout = timeout
	})
	return b
}

func (b *clientOptBuilder) MaxIdleConnsPerHost(conns int) *clientOptBuilder {
	b.setters = append(b.setters, func(co *clientOpt) {
		co.maxIdleConnsPerHost = conns
	})
	return b
}

func (b *clientOptBuilder) ResponseHeaderTimeout(timeout time.Duration) *clientOptBuilder {
	b.setters = append(b.setters, func(co *clientOpt) {
		co.responseHeaderTimeout = timeout
	})
	return b
}

func (b *clientOptBuilder) ServiceName(name string) *clientOptBuilder {
	b.setters = append(b.setters, func(co *clientOpt) {
		co.serviceName = name
	})
	return b
}

func (b *clientOptBuilder) Build() *clientOpt {
	args := &clientOpt{
		timeout:             http.DefaultClient.Timeout,
		maxIdleConnsPerHost: http.DefaultMaxIdleConnsPerHost,
	}

	for _, setter := range b.setters {
		setter(args)
	}
	return args
}
