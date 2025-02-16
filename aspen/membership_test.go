package aspen_test

import (
	"context"
	"github.com/synnaxlabs/x/address"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/aspen/mock"
	"go.uber.org/zap"
	"sync"
	"time"
)

var _ = Describe("Membership", Serial, Ordered, func() {
	var logger *zap.SugaredLogger

	BeforeEach(func() {
		l := zap.NewNop()
		logger = l.Sugar()
	})

	Describe("Bootstrap Cluster", func() {

		It("Should correctly bootstrap a cluster", func() {
			db, err := aspen.Open(
				context.TODO(),
				"",
				"localhost:22546",
				[]aspen.Address{},
				aspen.Bootstrap(),
				aspen.WithLogger(logger),
				aspen.MemBacked(),
			)

			By("Opening without error")
			Expect(err).ToNot(HaveOccurred())

			By("Assigning a valid NodeID of 1")
			Expect(db.HostID()).To(Equal(aspen.NodeID(1)))

			By("Adding itself to the node list")
			Expect(db.Nodes()).To(HaveLen(1))

			By("By setting its state to healthy")
			Expect(db.Host().State).To(Equal(aspen.Healthy))

			Expect(db.Close()).To(Succeed())
		})

		It("Should correctly bootstrap a cluster with peers provided", func() {
			db, err := aspen.Open(
				context.TODO(),
				"",
				"localhost:22546",
				[]aspen.Address{"localhost:22547"},
				aspen.WithLogger(logger),
				aspen.MemBacked(),
				aspen.Bootstrap(),
			)
			defer func() { Expect(db.Close()).To(Succeed()) }()

			By("Opening without error")
			Expect(err).ToNot(HaveOccurred())

			By("Assigning a valid NodeID of 1")
			Expect(db.HostID()).To(Equal(aspen.NodeID(1)))
		})

		It("Should correctly join a node that is already looking for peers", func() {
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
				defer cancel()
				db, err := aspen.Open(
					ctx,
					"",
					"localhost:22546",
					[]aspen.Address{"localhost:22547"},
					aspen.WithLogger(logger),
					aspen.MemBacked(),
				)
				defer func() { Expect(db.Close()).To(Succeed()) }()

				By("Joining the second node to the cluster without error")
				Expect(err).ToNot(HaveOccurred())

				By("Assigning a unique NodeID of 2")
				Expect(db.HostID()).To(Equal(aspen.NodeID(2)))
			}()
			db, err := aspen.Open(
				context.TODO(),
				"",
				"localhost:22547",
				[]aspen.Address{},
				aspen.WithLogger(logger),
				aspen.MemBacked(),
				aspen.Bootstrap(),
			)

			By("Joining the first node to the cluster without error")
			Expect(err).ToNot(HaveOccurred())

			By("Assigning a unique NodeID of 1")
			Expect(db.HostID()).To(Equal(aspen.NodeID(1)))
			wg.Wait()

			By("Safely closing the database")
			Expect(db.Close()).To(Succeed())
		})

	})

	Describe("Concurrent Pledges", func() {

		It("Should correctly join many nodes to the cluster concurrently", func() {
			numNodes := 10
			wg := sync.WaitGroup{}
			wg.Add(numNodes)
			var (
				addresses = address.NewLocalFactory(22546).NextN(numNodes)
				ids       = make([]aspen.NodeID, numNodes)
				dbs       = make([]aspen.DB, numNodes)
			)
			for i := 0; i < numNodes; i++ {
				go func(i int) {
					defer GinkgoRecover()
					defer wg.Done()
					opts := []aspen.Option{aspen.WithLogger(logger), aspen.MemBacked()}
					if i == 0 {
						opts = append(opts, aspen.Bootstrap())
					}
					db, err := aspen.Open(
						context.TODO(), "",
						addresses[i],
						addresses,
						opts...,
					)
					ids[i] = db.HostID()
					dbs[i] = db
					By("Joining the node to the cluster without error")
					Expect(err).ToNot(HaveOccurred())
				}(i)
			}
			wg.Wait()

			By("Assigning a unique NodeID to each node")
			Expect(len(lo.Uniq(ids))).To(Equal(len(ids)))

			By("Safely closing the database")
			for _, db := range dbs {
				Expect(db.Close()).To(Succeed())
			}
		})

	})

	Describe("Joining, Dying, and Rejoining", func() {
		Context("Persisted storage", func() {
			Context("Single node death", func() {
				It("Should correctly handle a single node dying and rejoining", func() {
					propConfig := aspen.PropagationConfig{
						PledgeRetryInterval:   10 * time.Millisecond,
						PledgeRetryScale:      1,
						ClusterGossipInterval: 50 * time.Millisecond,
					}
					builder := &mock.Builder{
						PortRangeStart: 22546,
						DataDir:        "./testdata",
						DefaultOptions: []aspen.Option{
							aspen.WithLogger(logger),
							aspen.WithPropagationConfig(propConfig),
						},
						Nodes: make(map[aspen.NodeID]mock.NodeInfo),
					}
					defer func() {
						Expect(builder.Cleanup()).To(Succeed())
					}()

					By("Forking the databases")
					for i := 0; i < 3; i++ {
						_, err := builder.New()
						Expect(err).ToNot(HaveOccurred())
					}

					By("Assigning the correct generation")
					node := builder.Nodes[2]
					Expect(node.DB.Host().Heartbeat.Generation).To(Equal(uint32(0)))

					By("Closing the database")
					Expect(node.DB.Close()).To(Succeed())

					By("Opening the database again")
					db, err := aspen.Open(
						context.TODO(),
						node.Dir,
						node.Addr,
						[]aspen.Address{},
						builder.DefaultOptions...,
					)
					Expect(err).ToNot(HaveOccurred())

					By("Assigning the correct NodeID")
					Expect(db.HostID()).To(Equal(aspen.NodeID(2)))

					By("Incrementing the heartbeat generation")
					Expect(db.Host().Heartbeat.Generation).To(Equal(uint32(1)))

					By("Propagating the incremented heartbeat to other nodes")
					ctx1 := builder.Nodes[1]
					Eventually(func(g Gomega) {
						n2, err := ctx1.DB.Node(2)
						g.Expect(err).ToNot(HaveOccurred())
						g.Expect(n2.State).To(Equal(aspen.Healthy))
						g.Expect(n2.Heartbeat.Generation).To(Equal(uint32(1)))
					})

					By("Closing the databases")
					Expect(builder.Nodes[1].DB.Close()).To(Succeed())
					Expect(builder.Nodes[3].DB.Close()).To(Succeed())
					Expect(db.Close()).To(Succeed())
				})
			})
		})
	})

})
