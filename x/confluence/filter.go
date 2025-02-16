package confluence

import (
	"context"
	"github.com/synnaxlabs/x/signal"
)

// Filter is a segment that reads values from an input Stream, filters them through a
// function, and optionally discards them to an output Stream.
type Filter[V Value] struct {
	AbstractLinear[V, V]
	FilterFunc[V]
	// Rejects is the Inlet that receives values that were discarded by Apply.
	Rejects Inlet[V]
}

type FilterFunc[V Value] struct {
	// Apply is called on each value passing through the Filter. If it returns false,
	// the value is discarded or sent to the Rejects Inlet. If it returns true,
	// the value is sent through the standard Inlet. If an error is returned,
	// the Filter is closed and a fatal error is returned to the context.
	Apply func(ctx context.Context, v V) (ok bool, err error)
}

// OutTo implements the Segment interface. It accepts either one or two Inlet(s).
// The first Inlet is where accepted values are sent, and the second Inlet (if provided)
// is where Rejected values are sent.
func (f *Filter[V]) OutTo(inlets ...Inlet[V]) {
	if len(inlets) > 2 || len(inlets) == 0 {
		panic("[confluence.ApplySink] - provide at most two and at least one inlet")
	}

	if len(inlets) == 1 {
		if f.AbstractLinear.Out != nil {
			f.Rejects = inlets[0]
			return
		}
	}

	f.AbstractLinear.OutTo(inlets[0])
	if len(inlets) == 2 {
		f.Rejects = inlets[1]
	}
}

// Flow implements the Segment interface.
func (f *Filter[V]) Flow(ctx signal.Context, opts ...Option) {
	fo := NewOptions(opts)
	fo.AttachClosables(f.Out, f.Rejects)
	f.GoRange(ctx, f.filter, fo.Signal...)
}

func (f *Filter[V]) filter(ctx context.Context, v V) error {
	ok, err := f.Apply(ctx, v)
	if err != nil {
		return err
	}
	if ok {
		return signal.SendUnderContext(ctx, f.Out.Inlet(), v)
	} else if f.Rejects != nil {
		return signal.SendUnderContext(ctx, f.Rejects.Inlet(), v)
	}
	return nil
}
