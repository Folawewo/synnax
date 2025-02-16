package signal

import (
	"context"
	"fmt"
	"runtime/pprof"
	"strconv"
)

type RoutineOption func(r *routine)

type RoutineInfo struct {
	// Key is a unique identifier for the routine within the parent context.
	Key string
	// ContextKey is the key of the context that the routine is running in.
	ContextKey string
	// State is the current state of the routine.
	State RoutineState
	// FailureReason is the error that caused the routine to exit.
	FailureReason error
}

func (r RoutineInfo) PrettyString() string {
	return fmt.Sprintf(
		`
ChannelKeys: %s
ContextKey: %s
State: %s
FailureReason: %s,
`, r.Key, r.ContextKey, r.State, r.FailureReason)
}

type RoutineState uint8

//go:generate stringer -type=RoutineState
const (
	Starting RoutineState = iota
	Running
	Stopping
	Exited
	Failed
	ContextCanceled
	Panicked
)

// Defer attaches the provided function f to the routine. The function will be
// executed on routine exit in LIFO style.
func Defer(f func()) RoutineOption {
	return func(r *routine) {
		r.deferals = append(r.deferals, deferal{key: "", f: f})
	}
}

func WithKey(key string) RoutineOption { return func(r *routine) { r.key = key } }

func WithKeyf(format string, args ...interface{}) RoutineOption {
	return func(r *routine) {
		r.key = fmt.Sprintf(format, args...)
	}
}

type deferal struct {
	key string
	f   func()
}

// CancelOnExit defines if the routine should cancel the context upon exiting.
// The default is false. If CancelOnExit or CancelOnExitErr has already been
// set, CancelOnExit will panic.
func CancelOnExit() RoutineOption {
	return func(r *routine) {
		if r.contextPolicy.cancelOnExit || r.contextPolicy.cancelOnExitErr {
			panic("[signal] - cannot set cancelOnExit or cancelOnExitErr twice")
		}
		r.contextPolicy.cancelOnExit = true
	}
}

// CancelOnExitErr defines if the routine should cancel the context upon exiting
// with a non-nil error. The default is false. If CancelOnExit or
// CancelOnExitErr has already been set, CancelOnExitErr will panic.
func CancelOnExitErr() RoutineOption {
	return func(r *routine) {
		if r.contextPolicy.cancelOnExit || r.contextPolicy.cancelOnExitErr {
			panic("[signal] - cannot set cancelOnExit or cancelOnExitErr twice")
		}
		r.contextPolicy.cancelOnExitErr = true
	}
}

type routine struct {
	ctx *core
	// key is a unique identifier for the routine. signal will panic if more
	// than one routine is started with the same key. If no key is provided
	// signal will automatically generate a unique key.
	key string
	// deferals is a list of functions to be called on routine exit in reverse
	// order.
	deferals []deferal
	// contextPolicy defines if the routine should cancel the context after
	// exiting.
	contextPolicy struct {
		cancelOnExit    bool
		cancelOnExitErr bool
	}
	// state represents the current state of the routine
	state struct {
		state RoutineState
		// err is the error that cause the routine to exit.
		err error
	}
}

func (r *routine) info() RoutineInfo {
	return RoutineInfo{
		Key:           r.key,
		ContextKey:    r.ctx.key,
		State:         r.state.state,
		FailureReason: r.state.err,
	}
}

func (r *routine) runPrelude() (proceed bool) {
	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()

	if r.key == "" {
		r.key = "anonymous-" + strconv.Itoa(len(r.ctx.mu.routines))
	}

	r.ctx.mu.routines = append(r.ctx.mu.routines, r)

	// If the context has already been canceled, don't even start the routine.
	if r.ctx.Err() != nil {
		r.state.state = Failed
		r.state.err = r.ctx.Err()
		return false
	}

	r.ctx.logger.Debugw("starting routine", r.diagnosticArgs()...)
	r.state.state = Running
	return true
}

func (r *routine) runPostlude(err error) {
	r.maybeRecover()
	defer r.maybeRecover()

	r.ctx.logger.Debugw("stopping routine", r.diagnosticArgs()...)

	r.ctx.mu.Lock()
	r.state.state = Stopping
	r.ctx.mu.Unlock()

	for i := range r.deferals {
		r.deferals[len(r.deferals)-i-1].f()
	}

	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()

	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			r.state.state = ContextCanceled
		} else {
			r.state.state = Failed
			r.state.err = err
			r.ctx.logger.Errorw("routine failed", r.diagnosticArgs()...)
			r.ctx.logger.Debugf(routineFailedFormat, r.key, r.state.err, r.ctx.routineDiagnostics())
		}
		if r.contextPolicy.cancelOnExitErr {
			r.ctx.cancel()
		}
	} else {
		r.state.state = Exited
	}

	if r.contextPolicy.cancelOnExit {
		r.ctx.cancel()
	}

	r.ctx.maybeStop()

	r.ctx.logger.Debugw("routine stopped", r.diagnosticArgs()...)
}

func (r *routine) maybeRecover() {
	if err := recover(); err != nil {
		r.ctx.mu.Lock()
		defer r.ctx.mu.Unlock()
		r.state.state = Panicked
		r.ctx.logger.Warnw("routine panicked")
		r.ctx.logger.Debugf(routineFailedFormat, r.key, err, r.ctx.routineDiagnostics())
		panic(err)
	}
}

func (r *routine) diagnosticArgs() []interface{} {
	deferalKeys := make([]string, len(r.deferals))
	for i, def := range r.deferals {
		deferalKeys[i] = def.key
	}

	return []interface{}{
		"key",
		r.key,
		"deferals",
		deferalKeys,
		"state",
		r.state.state,
	}
}

func (r *routine) goRun(f func(context.Context) error) {
	if !r.runPrelude() {
		return
	}
	profileKey := r.key
	if r.ctx.key != "" {
		profileKey = r.ctx.key + "." + r.key
	}
	labels := pprof.Labels("routine", profileKey)
	pprof.Do(r.ctx, labels, func(ctx context.Context) {
		r.ctx.wrapped.Go(func() (err error) {
			defer func() {
				r.runPostlude(err)
			}()
			err = f(ctx)
			return err
		})
	})
}

func newRoutine(c *core, opts []RoutineOption) *routine {
	r := &routine{ctx: c}
	for _, opt := range opts {
		opt(r)
	}

	return r
}

const routineFailedFormat = `
----------------------------- FAILURE DIAGNOSTICS ------------------------------

key: %s

--------------------------------- ERROR ----------------------------------------

%+v

-------------------------------- ROUTINES --------------------------------------
%s

----------------------------- END DIAGNOSTICS ----------------------------------
`
