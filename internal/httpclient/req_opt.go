package httpclient

type (
	reqOptBuilder struct {
		setters []func(*reqOpt)
	}

	reqOpt struct {
		canLog                      bool
		canLogRequestBody           bool
		canLogResponseBody          bool
		canLogRequestBodyOnlyError  bool
		canLogResponseBodyOnlyError bool
		loggedRequestBody           []string
		loggedResponseBody          []string
		markedQueryParamKeys        []string
		retryTimes                  uint
	}
)

func ReqOptBuilder() *reqOptBuilder {
	return &reqOptBuilder{}
}

func (b *reqOptBuilder) Log() *reqOptBuilder {
	b.setters = append(b.setters, func(ro *reqOpt) {
		ro.canLog = true
	})
	return b
}

func (b *reqOptBuilder) LogReqBody() *reqOptBuilder {
	b.setters = append(b.setters, func(ro *reqOpt) {
		ro.canLogRequestBody = true
	})
	return b
}

func (b *reqOptBuilder) LogResBody() *reqOptBuilder {
	b.setters = append(b.setters, func(ro *reqOpt) {
		ro.canLogResponseBody = true
	})
	return b
}

func (b *reqOptBuilder) LogReqBodyOnlyError() *reqOptBuilder {
	b.setters = append(b.setters, func(ro *reqOpt) {
		ro.canLogRequestBodyOnlyError = true
	})
	return b
}

func (b *reqOptBuilder) LogResBodyOnlyError() *reqOptBuilder {
	b.setters = append(b.setters, func(ro *reqOpt) {
		ro.canLogResponseBodyOnlyError = true
	})
	return b
}

func (b *reqOptBuilder) LoggedReqBody(body []string) *reqOptBuilder {
	b.setters = append(b.setters, func(ro *reqOpt) {
		ro.loggedRequestBody = body
	})
	return b
}

func (b *reqOptBuilder) LoggedResBody(body []string) *reqOptBuilder {
	b.setters = append(b.setters, func(ro *reqOpt) {
		ro.loggedResponseBody = body
	})
	return b
}

func (b *reqOptBuilder) MarkedQueryParamKeys(keys []string) *reqOptBuilder {
	b.setters = append(b.setters, func(ro *reqOpt) {
		ro.markedQueryParamKeys = keys
	})
	return b
}

func (b *reqOptBuilder) RetryTimes(retries uint) *reqOptBuilder {
	b.setters = append(b.setters, func(ro *reqOpt) {
		ro.retryTimes = retries
	})
	return b
}

func (b *reqOptBuilder) Build() *reqOpt {
	opt := &reqOpt{}
	for _, setter := range b.setters {
		setter(opt)
	}
	return opt
}
