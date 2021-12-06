# Private proxy docker registry

A private docker registry configured as a [pull through cache](https://docs.docker.com/registry/recipes/mirror/) allows to reduce the number of times docker images are downloaded from the Internet during the benchmarks, saving bandwidth and avoiding to incur in DockerHub rate limiting.
To this end, the testbed preparation scripts configure k3s to pull the images thorough the private docker registry.

A simple registry suitable for this scenario can be be set-up through the following Helm commands:

```bash
helm repo add twuni https://helm.twun.io
helm install docker-registry twuni/docker-registry \
    --namespace liqo-benchmarks --create-namespace --version=2.0.1  \
    --set persistence.enabled=true --set proxy.enabled=true
```
