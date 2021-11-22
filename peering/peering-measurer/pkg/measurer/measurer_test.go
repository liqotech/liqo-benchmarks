package measurer

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Measurer", func() {
	const id = "cluster-id"
	var (
		m  GlobalMeasurer
		ch chan struct{}

		t1, t2 time.Time
	)

	BeforeEach(func() {
		ch = make(chan struct{}, 1)
		m = *NewMeasurer(ch)

		t1 = time.Now()
		t2 = t1.Add(10 * time.Second)
	})

	It("Multiple calls to ClusterID should return the same object when the ID is the same", func() {
		a := m.ClusterID(id)
		b := m.ClusterID(id)
		Expect(a).To(BeIdenticalTo(b))
	})

	It("Multiple calls to ClusterID should return the same object when the ID is different", func() {
		a := m.ClusterID(id)
		b := m.ClusterID(id + "-second")
		Expect(a).ToNot(BeIdenticalTo(b))
	})

	It("Only the first call to SetAuthenticationIncomingEndTimestamp should take effect", func() {
		m.ClusterID(id).SetAuthenticationIncomingEndTimestamp(t1)
		m.ClusterID(id).SetAuthenticationIncomingEndTimestamp(t2)
		Expect(m.ClusterID(id).authenticationIncomingEnd).To(BeTemporally("~", t1, time.Second))
	})

	It("Only the first call to SetAuthenticationOutgoingEndTimestamp should take effect", func() {
		m.ClusterID(id).SetAuthenticationOutgoingEndTimestamp(t1)
		m.ClusterID(id).SetAuthenticationOutgoingEndTimestamp(t2)
		Expect(m.ClusterID(id).authenticationOutgoingEnd).To(BeTemporally("~", t1, time.Second))
	})

	It("Only the first call to SetResourceNegotiationStartTimestamp should take effect", func() {
		m.ClusterID(id).SetResourceNegotiationStartTimestamp(t1)
		m.ClusterID(id).SetResourceNegotiationStartTimestamp(t2)
		Expect(m.ClusterID(id).resourceNegotiationStart).To(BeTemporally("~", t1, time.Second))
	})

	It("Only the first call to SetResourceNegotiationEndTimestamp should take effect", func() {
		m.ClusterID(id).SetResourceNegotiationEndTimestamp(t1)
		m.ClusterID(id).SetResourceNegotiationEndTimestamp(t2)
		Expect(m.ClusterID(id).resourceNegotiationEnd).To(BeTemporally("~", t1, time.Second))
	})

	It("Only the first call to SetNetworkSetupStartTimestamp should take effect", func() {
		m.ClusterID(id).SetNetworkSetupStartTimestamp(t1)
		m.ClusterID(id).SetNetworkSetupStartTimestamp(t2)
		Expect(m.ClusterID(id).networkSetupStart).To(BeTemporally("~", t1, time.Second))
	})

	It("Only the first call to SetNetworkSetupEndTimestamp should take effect", func() {
		m.ClusterID(id).SetNetworkSetupEndTimestamp(t1)
		m.ClusterID(id).SetNetworkSetupEndTimestamp(t2)
		Expect(m.ClusterID(id).networkSetupEnd).To(BeTemporally("~", t1, time.Second))
	})

	It("Only the first call to SetVirtualKubeletSetupStartTimestamp should take effect", func() {
		m.ClusterID(id).SetVirtualKubeletSetupStartTimestamp(t1)
		m.ClusterID(id).SetVirtualKubeletSetupStartTimestamp(t2)
		Expect(m.ClusterID(id).virtualKubeletSetupStart).To(BeTemporally("~", t1, time.Second))
	})

	It("Only the first call to SetVirtualKubeletSetupEndTimestamp should take effect", func() {
		m.ClusterID(id).SetVirtualKubeletSetupEndTimestamp(t1)
		m.ClusterID(id).SetVirtualKubeletSetupEndTimestamp(t2)
		Expect(m.ClusterID(id).virtualKubeletSetupEnd).To(BeTemporally("~", t1, time.Second))
	})

	It("Only the first call to SetNodeReady should take effect", func() {
		m.ClusterID(id).SetNodeReady(t1)
		m.ClusterID(id).SetNodeReady(t2)
		Expect(m.ClusterID(id).nodeReady).To(BeTemporally("~", t1, time.Second))
	})

	It("SetNodeReady should correctly send a value to the channel", func() {
		m.ClusterID(id).SetNodeReady(time.Now())
		Expect(ch).To(Receive())
	})

	DescribeTable("The checkInterval function works correctly",
		func(delta time.Duration, expected bool) {
			a := time.Now()
			b := a.Add(delta)
			Expect(checkInterval(b, a)).To(Equal(expected))
		},
		Entry("When the difference is 0", 0*time.Second, true),
		Entry("When the difference is positive and small", 10*time.Millisecond, true),
		Entry("When the difference is positive and medium", 1400*time.Millisecond, true),
		Entry("When the difference is positive and large", 1600*time.Millisecond, false),
		Entry("When the difference is negative and small", -10*time.Millisecond, true),
		Entry("When the difference is negative and medium", -400*time.Millisecond, true),
		Entry("When the difference is negative and large", -600*time.Millisecond, false),
	)
})
