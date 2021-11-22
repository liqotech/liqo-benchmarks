#!/bin/bash

set -e
set -o pipefail

NAMESPACE="liqo-benchmarks"

echo "Retrieving the configuration parameters..."
echo "Namespace: $NAMESPACE"

KUBECTL="kubectl --namespace $NAMESPACE"

CONSUMER=$($KUBECTL get pod -l app.kubernetes.io/component=consumer --output custom-columns=':.metadata.name' --no-headers)
PROVIDER=$($KUBECTL get pod -l app.kubernetes.io/component=provider --output custom-columns=':.metadata.name' --no-headers)
PROVIDER_KUBECONF=$($KUBECTL get secret -l app.kubernetes.io/component=provider --output custom-columns=':.metadata.name' --no-headers)

CONSUMER_KUBECTL="$KUBECTL exec $CONSUMER -- kubectl"
PROVIDER_KUBECTL="$KUBECTL exec $PROVIDER -- kubectl"
CONSUMER_EXEC="$KUBECTL exec $CONSUMER -- /bin/sh -c"
PROVIDER_EXEC="$KUBECTL exec $PROVIDER -- /bin/sh -c"

echo "Configuring the metric server..."
$CONSUMER_KUBECTL patch deployment metrics-server \
  --namespace kube-system \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/args", "value": [
  "--metric-resolution=15s",
]}]'

# https://github.com/admiraltyio/admiralty/blob/master/docs/quick_start.md#cross-cluster-authentication
echo "Peering the admiralty clusters..."
$PROVIDER_KUBECTL create serviceaccount --namespace=default provider
SECRET_NAME=$($PROVIDER_KUBECTL get serviceaccount --namespace=default provider --template='{{ (index .secrets 0).name}}')
TOKEN=$($PROVIDER_KUBECTL get secret $SECRET_NAME --namespace=default --template='{{.data.token}}' | base64 --decode)
CONFIG=$(kubectl get secret $PROVIDER_KUBECONF --template='{{.data.kubeconfig}}' | base64 --decode | head -n -2)
CONFIG=$CONFIG$'\n    token: '$TOKEN
$CONSUMER_KUBECTL create secret generic --namespace=default provider --from-literal=config="$CONFIG"

$CONSUMER_EXEC 'cat <<EOF | kubectl apply -f -
apiVersion: multicluster.admiralty.io/v1alpha1
kind: Target
metadata:
  name: provider
spec:
  kubeconfigSecret:
    name: provider
EOF'

$PROVIDER_EXEC 'cat <<EOF | kubectl apply -f -
apiVersion: multicluster.admiralty.io/v1alpha1
kind: Source
metadata:
  name: provider
spec:
  serviceAccountName: provider
EOF'

echo "Waiting for the admiralty node to become ready..."
sleep 5 # Should be sufficient

echo "Enabling offloading on default namespace..."
$CONSUMER_KUBECTL label namespace default multicluster-scheduler=enabled

# Prevent offloaded pods from being scheduled on the k3s node
echo "Cordoning the provider node to force pods to be scheduled on hollow nodes..."
$PROVIDER_KUBECTL cordon "$PROVIDER"

echo
echo "Provider: $KUBECTL exec $PROVIDER -it -- /bin/sh"
echo "Consumer: $KUBECTL exec $CONSUMER -it -- /bin/sh"
