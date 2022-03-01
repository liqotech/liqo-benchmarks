# Peering benchmarks

This folder groups together the tools required to benchmark the liqo peering establishment process.

## liqo-k3s-cattle

[liqo-k3s-cattle](liqo-k3s-cattle) is an Helm chart that streamlines the creation of a given number of single-node K3s cluster on top of a pre-existing Kubernetes cluster.
This is meant to be leveraged as an easy-to-setup and reproducible platform for performance measurements.

Once started, each cluster installs and configures liqo, according to the specified values file.
Additional components can be enabled to automatically start the peering process and assess the completion time, as well as to measure the resources consumed (in terms of CPU, RAM and network traffic towards remote Kubernetes API servers) and possibly simulate additional network latency between the clusters.

To setup the testbed, it is sufficient to modify the [values](liqo-k3s-cattle/values.yaml) file according to the target configuration, and execute (from the current directory):

```bash
helm install liqo-k3s-cattle liqo-k3s-cattle --namespace liqo-benchmarks --create-namespace
```

Similarly, the testbed can be teared down through:

```bash
helm uninstall liqo-k3s-cattle --namespace liqo-benchmarks
```

## Peering measurer

[Peering measurer](peering-measurer) is a Go program responsible for identifying a set of target clusters, starting the liqo peering process (i.e., creating the appropriate ForeignCluster resources) and measuring the time required for process completion, breakdown by target cluster and sub-components.
It targets the super-cluster scenario (i.e., with a single hub cluster unidirectionally peering with a set of minion clusters) and it is meant to be executed on top of the liqo-k3s-cattle environment, as a pod hosted by the hub cluster.
To this end, the liqo-k3s-cattle Helm chart already includes the appropriate manifests to start the measurer with the correct parameters, depending on the values file configuration.

## Scripts

The [scripts](scripts) folder includes a set of scripts to automatically repeat the benchmark process with different configurations (e.g., number of minion clusters) and retrieve the resulting outcome, as well as to post-process the results and retrieve the most relevant information for subsequent analysis.
