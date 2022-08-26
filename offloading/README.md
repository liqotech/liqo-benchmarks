# Offloading benchmarks

This folder groups together the tools required to benchmark the liqo workload offloading and service exposition capabilities.

## liqo-k3s-hollow

[liqo-k3s-hollow](liqo-k3s-hollow) is an Helm chart that streamlines the creation of two K3s clusters (one playing the role of the resource provider and the other of the resource consumer) on top of a pre-existing Kubernetes cluster.
A configurable number of hollow nodes (cf. [kubemark](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-scalability/kubemark-guide.md)) is created and joined to the resource provider, to allow starting large amounts of (fake) pods with limited resource demands.
The overall environment is meant to be leveraged as an easy-to-setup and reproducible platform for performance measurements.

Once started, each cluster optionally installs and configures one multi-cluster solution among liqo, [Admiralty](https://github.com/admiraltyio/admiralty) and [tensile-kube](https://github.com/virtual-kubelet/tensile-kube), to enable the consumer to leverage the resources offered by the provider.
In case liqo is installed, the peering procedure between the two clusters can be also performed automatically.

To setup the testbed, it is sufficient to modify the [values](liqo-k3s-hollow/values.yaml) file according to the target configuration, and execute (from the current directory):

```bash
helm install liqo-k3s-hollow liqo-k3s-hollow --namespace liqo-benchmarks --create-namespace
```

Similarly, the testbed can be teared down through:

```bash
helm uninstall liqo-k3s-hollow --namespace liqo-benchmarks
```

## Offloading measurer

[Offloading measurer](offloading-measurer) is a Go program responsible for creating a set of Kubernetes deployments and measuring the time required for all pods to be created and become ready.
Specifically, it can be leveraged on top of the liqo-k3s-hollow environment to evaluate the pod offloading performance of liqo and alternative open-source solutions compared to vanilla Kubernetes.
Sample manifests to start the measurer in the different scenarios are available in the [manifests folder](scripts/manifests).

## Exposition measurer

[Exposition measurer](exposition-measurer) is a Go program responsible for creating a Kubernetes service and measuring the time required for all corresponding endpointslices to be created and become ready.
Specifically, it can be leveraged in combination with liqo-k3s-hollow and the offloading measurer to evaluate the service reflection performance of liqo compared to vanilla Kubernetes.
Sample manifests to start the measurer in the different scenarios are available in the [manifests folder](scripts/manifests).

## Exposition SYN measurer

[Exposition SYN measurer](exposition-syn-measurer) is a Go program that allows to measure the time elapsing from the creation of a Kubernetes service backed by a single endpoint up to the instant it is fully reachable.
This complements the comparison performed with the tool mentioned above, including both control plane (i.e., the object reflection performed by liqo) and data plane (i.e., iptables-related) aspects.
Technically speaking, it continuously probes the target service by means of TCP SYN segments, until the corresponding acknowledgement is received, confirming the reachability of the backend.
Sample manifests to start the measurer in the different scenarios are available in the [manifests folder](scripts/manifests).

## Storage measurer

[Storage measurer](storage-measurer) is a Go program responsible for creating a Kubernetes StatefulSet and measuring the time required for all pods to be created and become ready.
Specifically, it can be leveraged on top of the liqo-k3s-hollow environment (without hollow nodes) to evaluate the time required to create a new set of PVCs and bind them from a pod, when using either standard storage classes, or the virtual one featured by Liqo.
Sample manifests to start the measurer in the different scenarios are available in the [manifests folder](scripts/manifests).

## Scripts

The [scripts](scripts) folder includes a set of scripts to:

* Perform the appropriate operations to finalize the environment preparation, depending on the multi-cloud solution under test;
* Automatically repeat the benchmarks of interest with different configurations (e.g., number of pods to be created) and retrieve the resulting outcome.
* Post-process the results to retrieve the most relevant information for subsequent analysis.
