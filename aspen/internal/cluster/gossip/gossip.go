package gossip

import (
	"context"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/rand"
	"github.com/synnaxlabs/x/signal"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/aspen/internal/node"
	"time"
)

type Gossip struct{ Config }

// New opens a new Gossip that will spread cluster state to and from the given store.
func New(cfgs ...Config) (*Gossip, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	cfg.Logger.Infow("starting cluster gossip", cfg.Report().LogArgs()...)
	g := &Gossip{Config: cfg}
	alamos.AttachReporter(g.Experiment, "gossip", alamos.Debug, cfg)
	g.Transport.BindHandler(g.process)
	return g, nil
}

// GoGossip starts a goroutine that gossips at Config.Interval.
func (g *Gossip) GoGossip(ctx signal.Context) {
	g.Logger.Infow("starting periodic cluster gossip")
	signal.GoTick(
		ctx,
		g.Interval,
		func(ctx context.Context, t time.Time) error {
			if err := g.GossipOnce(ctx); err != nil {
				g.Logger.Errorw("gossip failed", "err", err)
			}
			return nil
		},
	)
}

func (g *Gossip) GossipOnce(ctx context.Context) error {
	g.incrementHostHeartbeat()
	snap := g.Store.CopyState()
	peer := RandomPeer(snap.Nodes, snap.HostID)
	if peer.Address == "" {
		return nil
	}
	return g.GossipOnceWith(ctx, peer.Address)

}

func (g *Gossip) GossipOnceWith(ctx context.Context, addr address.Address) error {
	sync := Message{Digests: g.Store.CopyState().Nodes.Digests()}
	ack, err := g.Transport.Send(ctx, addr, sync)
	if err != nil {
		return err
	}
	ack2 := g.ack(ack)
	if len(ack2.Nodes) == 0 {
		return nil
	}
	_, err = g.Transport.Send(ctx, addr, ack2)
	return err
}

func (g *Gossip) incrementHostHeartbeat() {
	host := g.Store.GetHost()
	host.Heartbeat = host.Heartbeat.Increment()
	g.Store.Set(host)
}

func (g *Gossip) process(ctx context.Context, msg Message) (Message, error) {
	switch msg.variant() {
	case messageVariantSync:
		return g.sync(msg), nil
	case messageVariantAck2:
		g.ack2(msg)
		return Message{}, nil
	}
	err := errors.New("[gossip] - received unknown message variant")
	g.Logger.Error(err, "msg", msg)
	return Message{}, err
}

func (g *Gossip) sync(sync Message) (ack Message) {
	snap := g.Store.CopyState()
	ack = Message{Nodes: make(node.Group), Digests: make(node.Digests)}
	for _, dig := range sync.Digests {
		n, ok := snap.Nodes[dig.ID]

		// If we have a more recent version of the node, return it to the initiator.
		if ok && n.Heartbeat.OlderThan(dig.Heartbeat) {
			ack.Nodes[dig.ID] = n
		}

		// If we don't have the node or our version is out of date, add it to our digests.
		if !ok || n.Heartbeat.YoungerThan(dig.Heartbeat) {
			ack.Digests[dig.ID] = node.Digest{ID: dig.ID, Heartbeat: n.Heartbeat}
		}
	}

	for _, n := range snap.Nodes {

		// If we have a node that the initiator doesn't have,
		// send it to them.
		if _, ok := sync.Digests[n.ID]; !ok {
			ack.Nodes[n.ID] = n
		}
	}

	return ack
}

func (g *Gossip) ack(ack Message) (ack2 Message) {
	// Take a snapshot before we merge the peer's nodes.
	snap := g.Store.CopyState()
	g.Store.Merge(ack.Nodes)
	ack2 = Message{Nodes: make(node.Group)}
	for _, dig := range ack.Digests {
		// If we have the node, and our version is newer, return it to the
		// peer.
		if n, ok := snap.Nodes[dig.ID]; ok && n.Heartbeat.OlderThan(dig.Heartbeat) {
			ack2.Nodes[dig.ID] = n
		}
	}
	return ack2
}

func (g *Gossip) ack2(ack2 Message) { g.Store.Merge(ack2.Nodes) }

func RandomPeer(nodes node.Group, host node.ID) node.Node {
	return rand.MapValue(nodes.WhereState(node.StateHealthy).WhereNot(host))
}
