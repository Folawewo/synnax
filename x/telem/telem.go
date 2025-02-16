package telem

import (
	"strconv"
	"time"
)

// |||||| TIME STAMP ||||||

const (
	// TimeStampMin represents the minimum value for a TimeStamp
	TimeStampMin = TimeStamp(0)
	// TimeStampMax represents the maximum value for a TimeStamp
	TimeStampMax = TimeStamp(^uint64(0) >> 1)
)

// TimeStamp stores an epoch time in nanoseconds.
type TimeStamp int64

// Now returns the current time as a TimeStamp.
func Now() TimeStamp { return NewTimeStamp(time.Now()) }

// NewTimeStamp creates a new TimeStamp from a time.Time.
func NewTimeStamp(t time.Time) TimeStamp { return TimeStamp(t.UnixNano()) }

// Time returns the time.Time representation of the TimeStamp.
func (ts TimeStamp) Time() time.Time { return time.Unix(0, int64(ts)) }

// IsZero returns true if the TimeStamp is TimeStampMin.
func (ts TimeStamp) IsZero() bool { return ts == TimeStampMin }

// After returns true if the TimeStamp is greater than the provided one.
func (ts TimeStamp) After(t TimeStamp) bool { return ts > t }

// AfterEq returns true if ts is less than or equal t t.
func (ts TimeStamp) AfterEq(t TimeStamp) bool { return ts >= t }

// Before returns true if the TimeStamp is less than the provided one.
func (ts TimeStamp) Before(t TimeStamp) bool { return ts < t }

// BeforeEq returns true if ts ie less than or equal to t.
func (ts TimeStamp) BeforeEq(t TimeStamp) bool { return ts <= t }

// Add returns a new TimeStamp with the provided TimeSpan added to it.
func (ts TimeStamp) Add(tspan TimeSpan) TimeStamp { return TimeStamp(int64(ts) + int64(tspan)) }

// Sub returns a new TimeStamp with the provided TimeSpan subtracted from it.
func (ts TimeStamp) Sub(tspan TimeSpan) TimeStamp { return TimeStamp(int64(ts) - int64(tspan)) }

// SpanRange constructs a new TimeRange with the TimeStamp and provided TimeSpan.
func (ts TimeStamp) SpanRange(span TimeSpan) TimeRange {
	rng := ts.Range(ts.Add(span))
	if !rng.Valid() {
		rng = rng.Swap()
	}
	return rng
}

// Range constructs a new TimeRange with the TimeStamp and provided TimeStamp.
func (ts TimeStamp) Range(ts2 TimeStamp) TimeRange { return TimeRange{ts, ts2} }

// Report implements fmt.Stringer.
//func (ts TimeStamp) Report() string { return ts.Time().Report() }

// |||||| TIME RANGE ||||||

// TimeRange represents a range of time between two TimeStamp. It's important
// to note that the start of the range is inclusive, while the end of hte range is
// exclusive.
type TimeRange struct {
	// Start is the start of the range.
	Start TimeStamp `json:"start" msgpack:"start"`
	// End is the end of the range.
	End TimeStamp `json:"end" msgpack:"end"`
}

// Span returns the TimeSpan that the TimeRange occupies.
func (tr TimeRange) Span() TimeSpan { return TimeSpan(tr.End - tr.Start) }

// IsZero returns true if the TimeSpan of TimeRange is empty.
func (tr TimeRange) IsZero() bool { return tr.Span().IsZero() }

// BoundBy limits the time range to the provided bounds.
func (tr TimeRange) BoundBy(otr TimeRange) TimeRange {
	if otr.Start.After(tr.Start) {
		tr.Start = otr.Start
	}
	if otr.Start.After(tr.End) {
		tr.End = otr.Start
	}
	if otr.End.Before(tr.End) {
		tr.End = otr.End
	}
	if otr.End.Before(tr.Start) {
		tr.Start = otr.End
	}
	return tr
}

// ContainsStamp returns true if the TimeRange contains the provided TimeStamp
func (tr TimeRange) ContainsStamp(stamp TimeStamp) bool {
	return stamp.AfterEq(tr.Start) && stamp.Before(tr.End)
}

// ContainsRange returns true if provided TimeRange contains the provided TimeRange.
// Returns true if the two ranges are equal.
func (tr TimeRange) ContainsRange(rng TimeRange) bool {
	return rng.Start.AfterEq(tr.Start) && rng.End.BeforeEq(tr.End)
}

// OverlapsWith returns true if the provided TimeRange overlaps with tr.
func (tr TimeRange) OverlapsWith(rng TimeRange) bool {
	if tr.End == rng.Start || tr.Start == rng.End {
		return false
	}
	return tr.ContainsStamp(rng.End) ||
		tr.ContainsStamp(rng.Start) ||
		rng.ContainsStamp(tr.Start) ||
		rng.ContainsStamp(tr.End)
}

func (tr TimeRange) Swap() TimeRange { return TimeRange{Start: tr.End, End: tr.Start} }

func (tr TimeRange) Valid() bool { return tr.Span() >= 0 }

var (
	// TimeRangeMax represents the maximum possible value for a TimeRange.
	TimeRangeMax = TimeRange{Start: TimeStampMin, End: TimeStampMax}
	// TimeRangeMin represents the minimum possible value for a TimeRange.
	TimeRangeMin = TimeRange{Start: TimeStampMax, End: TimeStampMin}
	// TimeRangeZero represents the zero value for a TimeRange.
	TimeRangeZero = TimeRange{Start: TimeStampMin, End: TimeStampMin}
)

// |||||| TIME SPAN ||||||

// TimeSpan represents a duration of time in nanoseconds.
type TimeSpan int64

const (
	// TimeSpanZero represents the zero value for a TimeSpan.
	TimeSpanZero = TimeSpan(0)
	// TimeSpanMax represents the maximum possible TimeSpan.
	TimeSpanMax = TimeSpan(^uint64(0) >> 1)
)

// Duration converts TimeSpan to a values.Duration.
func (ts TimeSpan) Duration() time.Duration { return time.Duration(ts) }

// Seconds returns a float64 value representing the number of seconds in the TimeSpan.
func (ts TimeSpan) Seconds() float64 { return ts.Duration().Seconds() }

// IsZero returns true if the TimeSpan is TimeSpanZero.
func (ts TimeSpan) IsZero() bool { return ts == TimeSpanZero }

// IsMax returns true if the TimeSpan is the maximum possible value.
func (ts TimeSpan) IsMax() bool { return ts == TimeSpanMax }

func (ts TimeSpan) ByteSize(rate Rate, density Density) Size {
	return Size(ts / rate.Period() * TimeSpan(density))
}

const (
	Nanosecond  = TimeSpan(1)
	Microsecond = 1000 * Nanosecond
	Millisecond = 1000 * Microsecond
	Second      = 1000 * Millisecond
	Minute      = 60 * Second
	Hour        = 60 * Minute
)

// |||||| SIZE ||||||

// Size represents the size of an element in bytes.
type Size int64

type Offset = Size

const Kilobytes Size = 1024

// String implements fmt.Stringer.
func (s Size) String() string { return strconv.Itoa(int(s)) + "B" }

// |||||| DATA RATE ||||||

// Rate represents a rate in Hz.
type Rate float64

// Period returns a TimeSpan representing the period of the Rate.
func (dr Rate) Period() TimeSpan { return TimeSpan(1 / float64(dr) * float64(Second)) }

// SampleCount returns n integer representing the number of samples in the provided Span.
func (dr Rate) SampleCount(t TimeSpan) int { return int(t.Seconds() * float64(dr)) }

// Span returns a TimeSpan representing the number of samples that occupy the provided Span.
func (dr Rate) Span(sampleCount int) TimeSpan { return dr.Period() * TimeSpan(sampleCount) }

// SizeSpan returns a TimeSpan representing the number of samples that occupy a provided number of bytes.
func (dr Rate) SizeSpan(size Size, Density Density) TimeSpan {
	return dr.Span(int(size) / int(Density))
}

const (
	// Hz represents a data rate of 1 Hz.
	Hz  Rate = 1
	KHz      = 1000 * Hz
	MHz      = 1000 * KHz
)

// |||||| DENSITY ||||||

type (
	// Density represents the density of a data type in bytes per sample.
	Density uint32
)

const (
	DensityUnknown Density = 0
	Bit64          Density = 8
	Bit32          Density = 4
	Bit16          Density = 2
	Bit8           Density = 1
)
