package mock

import (
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/aspen/internal/node"
	"go/types"
)

type Network struct {
	pledge     *fmock.Network[node.ID, node.ID]
	cluster    *fmock.Network[gossip.Message, gossip.Message]
	operations *fmock.Network[kv.BatchRequest, kv.BatchRequest]
	lease      *fmock.Network[kv.BatchRequest, types.Nil]
	feedback   *fmock.Network[kv.FeedbackMessage, types.Nil]
}

func NewNetwork() *Network {
	return &Network{
		pledge:     fmock.NewNetwork[node.ID, node.ID](),
		cluster:    fmock.NewNetwork[gossip.Message, gossip.Message](),
		operations: fmock.NewNetwork[kv.BatchRequest, kv.BatchRequest](),
		lease:      fmock.NewNetwork[kv.BatchRequest, types.Nil](),
		feedback:   fmock.NewNetwork[kv.FeedbackMessage, types.Nil](),
	}
}

func (n *Network) NewTransport() aspen.Transport { return &transport{net: n} }

// transport is an in-memory, synchronous implementation of aspen.transport.
type transport struct {
	net        *Network
	pledge     *fmock.Unary[node.ID, node.ID]
	cluster    *fmock.Unary[gossip.Message, gossip.Message]
	operations *fmock.Unary[kv.BatchRequest, kv.BatchRequest]
	lease      *fmock.Unary[kv.BatchRequest, types.Nil]
	feedback   *fmock.Unary[kv.FeedbackMessage, types.Nil]
}

// Configure implements aspen.transport.
func (t *transport) Configure(ctx signal.Context, addr address.Address, external bool) error {
	t.pledge = t.net.pledge.RouteUnary(addr)
	t.cluster = t.net.cluster.RouteUnary(addr)
	t.operations = t.net.operations.RouteUnary(addr)
	t.lease = t.net.lease.RouteUnary(addr)
	t.feedback = t.net.feedback.RouteUnary(addr)
	return nil
}

// Pledge implements aspen.transport.
func (t *transport) Pledge() pledge.Transport { return t.pledge }

// Cluster implements aspen.transport.
func (t *transport) Cluster() gossip.Transport { return t.cluster }

// Operations implements aspen.transport.
func (t *transport) Operations() kv.BatchTransport { return t.operations }

// Lease implements aspen.transport.
func (t *transport) Lease() kv.LeaseTransport { return t.lease }

// Feedback implements aspen.transport.
func (t *transport) Feedback() kv.FeedbackTransport { return t.feedback }
