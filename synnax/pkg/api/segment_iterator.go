package api

import (
	"context"
	roacherrors "github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/iterator"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type (
	IteratorCommand         = iterator.Command
	IteratorResponseVariant = iterator.ResponseVariant
)

type IteratorRequest struct {
	Command IteratorCommand `json:"command" msgpack:"command"`
	Span    telem.TimeSpan  `json:"span" msgpack:"span"`
	Range   telem.TimeRange `json:"range" msgpack:"range"`
	Stamp   telem.TimeStamp `json:"stamp" msgpack:"stamp"`
	Keys    []string        `json:"keys" msgpack:"keys"`
}

type IteratorResponse struct {
	Variant  IteratorResponseVariant `json:"variant" msgpack:"variant"`
	Command  IteratorCommand         `json:"command" msgpack:"command"`
	Ack      bool                    `json:"ack" msgpack:"ack"`
	Err      ferrors.Payload         `json:"error" msgpack:"error"`
	Segments []Segment               `json:"segments" msgpack:"segments"`
}

type IteratorStream = freighter.ServerStream[IteratorRequest, IteratorResponse]

func (s *SegmentService) Iterate(_ctx context.Context, stream IteratorStream) errors.Typed {
	ctx, cancel := signal.WithCancel(_ctx, signal.WithLogger(s.Logger.Desugar()))
	// cancellation here would occur for one of two reasons. Either we encounter
	// a fatal error (transport or iterator internal) and we need to free all
	// resources, OR the client executed the close command on the iterator (in
	// which case resources have already been freed and cancel does nothing).
	defer cancel()

	iter, err := s.openIterator(ctx, stream)
	if err.Occurred() {
		return errors.Unexpected(err)
	}
	requests := confluence.NewStream[iterator.Request]()
	iter.InFrom(requests)
	responses := confluence.NewStream[iterator.Response]()
	iter.OutTo(responses)
	iter.Flow(ctx, confluence.CloseInletsOnExit(), confluence.CancelOnExitErr())

	go func() {
		for {
			req, err := stream.Receive()
			if roacherrors.Is(err, freighter.EOF) {
				requests.Close()
				return
			}
			if err != nil {
				return
			}
			requests.Inlet() <- iterator.Request{
				Command: req.Command,
				Span:    req.Span,
				Range:   req.Range,
				Stamp:   req.Stamp,
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return errors.Canceled
		case res, ok := <-responses.Outlet():
			if !ok {
				return errors.Nil
			}
			segments := make([]Segment, len(res.Segments))
			for i, seg := range res.Segments {
				segments[i] = Segment{
					ChannelKey: seg.ChannelKey.String(),
					Start:      seg.Segment.Start,
					Data:       seg.Segment.Data,
				}
			}
			tRes := IteratorResponse{
				Variant:  res.Variant,
				Command:  res.Command,
				Ack:      res.Ack,
				Segments: segments,
			}
			if res.Error != nil {
				tRes.Err = ferrors.Encode(res.Error)
			}
			if err := stream.Send(tRes); err != nil {
				return errors.Unexpected(err)
			}
		}
	}
}

func (s *SegmentService) openIterator(ctx context.Context, srv IteratorStream) (segment.StreamIterator, errors.Typed) {
	keys, rng, _err := receiveIteratorOpenArgs(srv)
	if _err.Occurred() {
		return nil, _err
	}
	iter, err := s.Internal.NewStreamIterator(ctx, rng, keys...)
	if err != nil {
		return nil, errors.Query(err)
	}
	return iter, errors.MaybeUnexpected(srv.Send(IteratorResponse{Variant: iterator.AckResponse, Ack: true}))
}

func receiveIteratorOpenArgs(srv IteratorStream) (channel.Keys, telem.TimeRange, errors.Typed) {
	req, err := srv.Receive()
	if err != nil {
		return nil, telem.TimeRangeZero, errors.Unexpected(err)
	}
	if req.Command != iterator.Open {
		return nil, telem.TimeRangeZero, errors.Parse(roacherrors.New("expected open command"))
	}
	keys, err := channel.ParseKeys(req.Keys)

	if req.Range.IsZero() {
		req.Range = telem.TimeRangeMax
	}

	return keys, req.Range, errors.MaybeParse(err)
}
