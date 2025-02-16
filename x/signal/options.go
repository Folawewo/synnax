package signal

import "go.uber.org/zap"

// |||||| OPTIONS ||||||

type Option func(o *options)

func WithLogger(logger *zap.Logger) Option {
	return func(o *options) { o.logger = logger.Named("signal").Sugar() }
}

func WithContextKey(key string) Option {
	return func(o *options) { o.key = key }
}

type options struct {
	key    string
	logger *zap.SugaredLogger
}

func newOptions(opts ...Option) *options {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	mergeDefaultOptions(o)
	return o
}

func mergeDefaultOptions(o *options) {
	if o.logger == nil {
		o.logger = zap.NewNop().Sugar()
	}
}
