package iterator_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/testutil/seg"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/iterator"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
	"testing"
)

var (
	ctx = context.Background()
)

func TestIterator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Iterator Suite")
}

type serviceContainer struct {
	channel   *channel.Service
	transport struct {
		channel channel.CreateTransport
		iter    iterator.Transport
	}
}

func provisionNServices(n int, logger *zap.Logger) (*mock.CoreBuilder, map[core.NodeID]serviceContainer) {
	builder := mock.NewCoreBuilder(core.Config{Logger: logger, Storage: storage.Config{MemBacked: config.BoolPointer(true)}})
	services := make(map[core.NodeID]serviceContainer)
	channelNet := fmock.NewNetwork[channel.CreateMessage, channel.CreateMessage]()
	iterNet := fmock.NewNetwork[iterator.Request, iterator.Response]()
	for i := 0; i < n; i++ {
		_core := builder.New()
		var container serviceContainer
		container.transport.channel = channelNet.RouteUnary(_core.Config.AdvertiseAddress)
		container.transport.iter = iterNet.RouteStream(_core.Config.AdvertiseAddress, 0)
		container.channel = channel.New(
			_core.Cluster,
			_core.Storage.Gorpify(),
			_core.Storage.TS,
			container.transport.channel,
		)
		iterator.NewServer(iterator.Config{
			TS:        _core.Storage.TS,
			Resolver:  _core.Cluster,
			Transport: container.transport.iter,
			Logger:    zap.NewNop(),
		})
		services[_core.Cluster.HostID()] = container
	}
	return builder, services
}

func writeMockData(
	builder *mock.CoreBuilder,
	segmentSize telem.TimeSpan,
	numberOfRequests, numberOfSegmentsPerRequest int,
	channels ...channel.Channel,
) {
	dataFactory := &seg.RandomFloat64Factory{Cache: true}
	for _, ch := range channels {
		factory := seg.NewSequentialFactory(dataFactory, segmentSize, ch.Channel)
		Expect(builder.Cores[ch.NodeID].
			Storage.
			TS.
			Write(factory.NextN(numberOfSegmentsPerRequest * numberOfRequests))).To(Succeed())
	}
}

func openIter(
	nodeID core.NodeID,
	services map[core.NodeID]serviceContainer,
	builder *mock.CoreBuilder,
	keys channel.Keys,
) iterator.Iterator {
	iter, err := iterator.New(
		ctx,
		iterator.Config{
			Logger:         zap.NewNop(),
			TS:             builder.Cores[nodeID].Storage.TS,
			Resolver:       builder.Cores[nodeID].Cluster,
			Transport:      services[nodeID].transport.iter,
			TimeRange:      telem.TimeRangeMax,
			ChannelKeys:    keys,
			ChannelService: services[nodeID].channel,
		},
	)
	Expect(err).ToNot(HaveOccurred())
	return iter
}
