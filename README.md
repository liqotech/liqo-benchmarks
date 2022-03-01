# liqo-benchmarks

This repository contains a set of tools to streamline the benchmarking of [liqo](https://github.com/liqotech/liqo/) along the following verticals:

* *Peering establishment*, to assess the scalability of the process and evaluate the time elapsing from the discovery of a new peering candidate to the creation of the corresponding virtual node, while varying the number of target clusters.

* *Application offloading*, to analyze the capability to start a huge burst of pods (offloaded to a remote cluster), compared to alternative virtual-kubelet based open-source projects, such as [Admiralty](https://github.com/admiraltyio/admiralty) and [tensile-kube](https://github.com/virtual-kubelet/tensile-kube).

* *Service exposition*, to evaluate the time required by liqo to replicate a service and all the associated endpointslices to a remote cluster, making them available for consumption by remote applications.

* *Resource consumption*, to measure the liqo resource demands, in terms of CPU and RAM required by the liqo control plane, as well as the network traffic generated towards remote Kubernetes API servers.

Further details about the testbed setup and the benchmark execution are provided in the respective sub-folders.
The [various](./various) folder groups together additional tools unrelated from any specific benchmark.
