package confluence_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Delta", func() {
	var (
		inputOne  Stream[int]
		outputOne Stream[int]
		outputTwo Stream[int]
	)
	BeforeEach(func() {
		inputOne = NewStream[int](1)
		outputOne = NewStream[int](0)
		outputOne.SetInletAddress("outputOne")
		outputTwo = NewStream[int](0)
		outputTwo.SetInletAddress("outputTwo")
	})
	Describe("DeltaMultiplier", func() {
		It("Should multiply input values to outputs", func() {
			delta := &DeltaMultiplier[int]{}
			delta.OutTo(outputOne, outputTwo)
			delta.InFrom(inputOne)
			ctx, cancel := signal.TODO()
			defer cancel()
			delta.Flow(ctx)
			inputOne.Inlet() <- 1
			v1 := <-outputOne.Outlet()
			v2 := <-outputTwo.Outlet()
			Expect(v1).To(Equal(1))
			Expect(v2).To(Equal(1))
		})
		It("Should close inlets when the delta is closed", func() {
			delta := &DeltaMultiplier[int]{}
			delta.OutTo(outputOne)
			delta.InFrom(inputOne)
			ctx, cancel := signal.TODO()
			defer cancel()
			delta.Flow(ctx, CloseInletsOnExit())
			inputOne.Inlet() <- 1
			inputOne.Close()
			v1 := <-outputOne.Outlet()
			Expect(v1).To(Equal(1))
			_, ok := <-outputOne.Outlet()
			Expect(ok).To(BeFalse())
		})
	})
	Describe("DeltaTransformMultiplier", func() {
		It("Should multiply input values to outputs", func() {
			delta := &DeltaTransformMultiplier[int, int]{}
			delta.Transform = func(ctx context.Context, v int) (int, bool, error) {
				return v * 2, true, nil
			}
			delta.OutTo(outputOne, outputTwo)
			delta.InFrom(inputOne)
			ctx, cancel := signal.TODO()
			defer cancel()
			delta.Flow(ctx)
			inputOne.Inlet() <- 1
			v1 := <-outputOne.Outlet()
			v2 := <-outputTwo.Outlet()
			Expect(v1).To(Equal(2))
			Expect(v2).To(Equal(2))

		})
		It("Should close inlets when the delta is closed", func() {
			delta := &DeltaTransformMultiplier[int, int]{}
			delta.Transform = func(ctx context.Context, v int) (int, bool, error) {
				return v * 2, true, nil
			}
			delta.OutTo(outputOne)
			delta.InFrom(inputOne)
			ctx, cancel := signal.TODO()
			defer cancel()
			delta.Flow(ctx, CloseInletsOnExit())
			inputOne.Inlet() <- 1
			inputOne.Close()
			v1 := <-outputOne.Outlet()
			Expect(v1).To(Equal(2))
			_, ok := <-outputOne.Outlet()
			Expect(ok).To(BeFalse())
		})
		It("Should not send a value when the transform returns false", func() {
			delta := &DeltaTransformMultiplier[int, int]{}
			delta.Transform = func(ctx context.Context, v int) (int, bool, error) {
				return v * 2, v != 1, nil
			}
			delta.OutTo(outputOne)
			delta.InFrom(inputOne)
			ctx, cancel := signal.TODO()
			defer cancel()
			delta.Flow(ctx, CloseInletsOnExit())
			inputOne.Inlet() <- 1
			inputOne.Close()
			_, ok := <-outputOne.Outlet()
			Expect(ok).To(BeFalse())
		})
	})
	Describe("DynamicDeltaMultiplier", func() {
		It("Should allow the caller to add and remove outlets dynamically", func() {
			delta := NewDynamicDeltaMultiplier[int]()
			delta.InFrom(inputOne)
			ctx, cancel := signal.TODO()
			defer cancel()
			delta.Flow(ctx, CloseInletsOnExit())
			delta.Connect(outputOne)
			delta.Connect(outputTwo)
			inputOne.Inlet() <- 1
			v1 := <-outputOne.Outlet()
			v2 := <-outputTwo.Outlet()
			Expect(v1).To(Equal(1))
			Expect(v2).To(Equal(1))
			delta.Disconnect(outputOne)
			inputOne.Inlet() <- 2
			v2 = <-outputTwo.Outlet()
			_, ok := <-outputOne.Outlet()
			Expect(v2).To(Equal(2))
			Expect(ok).To(BeFalse())
		})
	})
})
