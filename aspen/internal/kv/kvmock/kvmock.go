package kvmock

import (
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/clustermock"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/aspen/internal/node"
	"go/types"
)

type Builder struct {
	clustermock.Builder
	BaseCfg     kv.Config
	OpNet       *fmock.Network[kv.BatchRequest, kv.BatchRequest]
	FeedbackNet *fmock.Network[kv.FeedbackMessage, types.Nil]
	LeaseNet    *fmock.Network[kv.BatchRequest, types.Nil]
	KVs         map[node.ID]kv.DB
}

func NewBuilder(baseKVCfg kv.Config, baseClusterCfg cluster.Config) *Builder {
	return &Builder{
		BaseCfg:     baseKVCfg,
		Builder:     *clustermock.NewBuilder(baseClusterCfg),
		OpNet:       fmock.NewNetwork[kv.BatchRequest, kv.BatchRequest](),
		FeedbackNet: fmock.NewNetwork[kv.FeedbackMessage, types.Nil](),
		LeaseNet:    fmock.NewNetwork[kv.BatchRequest, types.Nil](),
		KVs:         make(map[node.ID]kv.DB),
	}
}

func (b *Builder) New(ctx signal.Context, kvCfg kv.Config, clusterCfg cluster.Config) (kv.DB, error) {
	clust, err := b.Builder.New(ctx, clusterCfg)
	if err != nil {
		return nil, err
	}
	kvCfg = b.BaseCfg.Override(kvCfg)
	if kvCfg.Engine == nil {
		kvCfg.Engine = memkv.New()
	}
	kvCfg.Cluster = clust
	addr := clust.Host().Address
	kvCfg.OperationsTransport = b.OpNet.RouteUnary(addr)
	kvCfg.FeedbackTransport = b.FeedbackNet.RouteUnary(addr)
	kvCfg.LeaseTransport = b.LeaseNet.RouteUnary(addr)
	kve, err := kv.Open(ctx, kvCfg)
	if err != nil {
		return nil, err
	}
	b.KVs[clust.Host().ID] = kve
	return kve, nil
}
