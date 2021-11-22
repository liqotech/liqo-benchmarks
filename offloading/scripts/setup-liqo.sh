#!/bin/bash

set -e
set -o pipefail

NAMESPACE="liqo-benchmarks"

echo "Retrieving the configuration parameters..."
echo "Namespace: $NAMESPACE"

KUBECTL="kubectl --namespace $NAMESPACE"
CONSUMER=$($KUBECTL get pod -l app.kubernetes.io/component=consumer --output custom-columns=':.metadata.name' --no-headers)
PROVIDER=$($KUBECTL get pod -l app.kubernetes.io/component=provider --output custom-columns=':.metadata.name' --no-headers)

CONSUMER_KUBECTL="$KUBECTL exec $CONSUMER -- kubectl"
PROVIDER_KUBECTL="$KUBECTL exec $PROVIDER -- kubectl"

echo "Configuring the metric server..."
$CONSUMER_KUBECTL patch deployment metrics-server \
  --namespace kube-system \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/args", "value": [
  "--metric-resolution=15s",
]}]'

# Prevent offloaded pods from being scheduled on the k3s node
echo "Cordoning the provider node to force pods to be scheduled on hollow nodes..."
$PROVIDER_KUBECTL cordon "$PROVIDER"

echo
echo "Provider: $KUBECTL exec $PROVIDER -it -- /bin/sh"
echo "Consumer: $KUBECTL exec $CONSUMER -it -- /bin/sh"
