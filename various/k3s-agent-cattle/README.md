# k3s-agent-cattle

`k3s-agent-cattle` is an Helm chart that streamlines the creation of a K3s cluster, composed of one server and a given numbers of agents, on top of a pre-existing Kubernetes cluster.
This is meant to be leveraged as an easy-to-setup and reproducible platform for performance measurements.

To setup the testbed, it is sufficient to modify the [values](values.yaml) file according to the target configuration, and execute (from the current directory):

```bash
helm install k3s-agent-cattle . --namespace liqo-benchmarks --create-namespace
```

Similarly, the testbed can be teared down through:

```bash
helm uninstall k3s-agent-cattle --namespace liqo-benchmarks
```
