package observe_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
	"time"
)

type dataStruct struct {
	Value []byte
}

var _ = Describe("Flush", func() {
	It("Should flush the observable contents", func() {
		o := observe.New[dataStruct]()
		kv := memkv.New()
		ecd := &binary.GobEncoderDecoder{}
		flush := &observe.FlushSubscriber[dataStruct]{
			Key:         []byte("key"),
			Store:       kv,
			MinInterval: 5 * time.Millisecond,
			Encoder:     ecd,
		}
		o.OnChange(flush.Flush)

		o.Notify(dataStruct{Value: []byte("hello")})
		o.Notify(dataStruct{Value: []byte("world")})

		Eventually(func(g Gomega) {
			b, err := kv.Get([]byte("key"))
			g.Expect(err).ToNot(HaveOccurred())
			var ds dataStruct
			g.Expect(ecd.Decode(b, &ds)).To(Succeed())
			g.Expect(ds.Value).To(Equal([]byte("hello")))
		}).Should(Succeed())
	})
})
