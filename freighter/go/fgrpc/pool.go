package fgrpc

import (
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// ClientConn is a wrapper around grpc.ClientConn that implements the
// pool.Adapter interface.
type ClientConn struct {
	*grpc.ClientConn
	demand *pool.Demand
}

// Acquire implements pool.Adapter.
func (c *ClientConn) Acquire() error {
	c.demand.Increase(1)
	return nil
}

// Release implements pool.Adapter.
func (c *ClientConn) Release() {
	c.demand.Decrease(1)
}

// Close implements pool.Adapter.
func (c *ClientConn) Close() error { return c.ClientConn.Close() }

// Healthy implements pool.Adapter
func (c *ClientConn) Healthy() bool {
	state := c.GetState()
	return state != connectivity.TransientFailure && state != connectivity.Shutdown
}

type Pool struct {
	pool.Pool[address.Address, *ClientConn]
}

func NewPool(dialOpts ...grpc.DialOption) *Pool {
	return &Pool{Pool: pool.New[address.Address, *ClientConn](&factory{dialOpts: dialOpts})}
}

// factory implements the pool.Factory interface.
type factory struct {
	dialOpts []grpc.DialOption
}

func (f *factory) New(addr address.Address) (*ClientConn, error) {
	c, err := grpc.Dial(string(addr), f.dialOpts...)
	d := pool.Demand(1)
	return &ClientConn{ClientConn: c, demand: &d}, err
}
