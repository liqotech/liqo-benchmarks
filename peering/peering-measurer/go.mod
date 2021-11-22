module github.com/liqotech/liqo-benchmarks/peering/peering-measurer

go 1.16

require (
	github.com/liqotech/liqo v0.3.2-alpha.3.0.20211122105130-beaf6ae0219e
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/klog/v2 v2.10.0
)

replace github.com/grandcat/zeroconf => github.com/liqotech/zeroconf v1.0.1-0.20201020081245-6384f3f21ffb
