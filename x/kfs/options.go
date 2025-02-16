package kfs

import (
	"github.com/synnaxlabs/x/alamos"
	"os"
	"time"
)

type options struct {
	baseFS          BaseFS
	suffix          string
	experiment      alamos.Experiment
	maxSyncInterval time.Duration
	dirPerms        os.FileMode
}

type Option func(o *options)

func newOptions(opts ...Option) *options {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	mergeDefaultOptions(o)
	return o
}

const defaultSuffix = ".kfs"

func mergeDefaultOptions(o *options) {
	if o.suffix == "" {
		o.suffix = defaultSuffix
	}
	if o.baseFS == nil {
		o.baseFS = &osFS{}
	}
	if o.dirPerms == 0 {
		o.dirPerms = 0777
	}
}

// WithFS sets the base filesystem to use.
func WithFS(fs BaseFS) Option {
	return func(o *options) {
		o.baseFS = fs
	}
}

// WithExperiment sets the experiment that the KFS uses to record its Metrics.
func WithExperiment(e alamos.Experiment) Option {
	return func(o *options) {
		o.experiment = e
	}
}

// WithExtensionConfig sets the suffix that the KFS uses to append to its filenames.
func WithExtensionConfig(s string) Option {
	return func(o *options) {
		o.suffix = s
	}
}

// WithDirPerms sets the permissions that the KFS uses to create directories.
func WithDirPerms(perms os.FileMode) Option {
	return func(o *options) {
		o.dirPerms = perms
	}
}
