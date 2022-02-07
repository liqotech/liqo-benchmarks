#!/bin/bash

set -e
set -o pipefail

if [ $# -ne 3 ]; then
    echo "Usage: $0 exposition-manifest-local exposition-manifest-remote output-folder"
    echo "Measurer manifest not provided"
    exit 1
fi

EXPOSITION_MANIFEST_LOCAL=$1
EXPOSITION_MANIFEST_LOCAL_FILE=$(basename "$EXPOSITION_MANIFEST_LOCAL")
EXPOSITION_MANIFEST_REMOTE=$2
EXPOSITION_MANIFEST_REMOTE_FILE=$(basename "$EXPOSITION_MANIFEST_REMOTE")
OUTPUT=$3

RUNS=10

NAMESPACE="liqo-benchmarks"

echo "Retrieving the configuration parameters..."
echo "Namespace: $NAMESPACE"
KUBECTL="kubectl --namespace $NAMESPACE"
CONSUMER=$($KUBECTL get pod -l app.kubernetes.io/component=consumer --output custom-columns=':.metadata.name' --no-headers)
PROVIDER=$($KUBECTL get pod -l app.kubernetes.io/component=provider --output custom-columns=':.metadata.name' --no-headers)
PROVIDER_EXEC="$KUBECTL exec $PROVIDER -c k3s-server -- /bin/sh"
CONSUMER_KUBECTL="$KUBECTL exec $CONSUMER -c k3s-server -- kubectl"
PROVIDER_KUBECTL="$KUBECTL exec $PROVIDER -c k3s-server -- kubectl"


echo "Copying the measurer manifests to the provider..."
tar cf - -C "$(dirname $EXPOSITION_MANIFEST_LOCAL)" "$(basename $EXPOSITION_MANIFEST_LOCAL)" | \
    $KUBECTL exec "$PROVIDER" -c k3s-server -i -- tar xf - -C "/tmp"

echo "Copying the measurer manifests to the consumer..."
tar cf - -C "$(dirname $EXPOSITION_MANIFEST_REMOTE)" "$(basename $EXPOSITION_MANIFEST_REMOTE)" | \
    $KUBECTL exec "$CONSUMER" -c k3s-server -i -- tar xf - -C "/tmp"

mkdir --parent "$OUTPUT"
echo "Ready to start executing the benchmarks"
for RUN in $(seq 1 $RUNS); do
    echo
    echo "Run $RUN"

    echo "Starting the nginx pod in the provider"
    $PROVIDER_KUBECTL create namespace exposition-benchmark
    $PROVIDER_KUBECTL run nginx --image=nginx:1.21 --namespace=exposition-benchmark \
        --labels=app.kubernetes.io/part-of=benchmarks
    $PROVIDER_KUBECTL wait --timeout=-1s --namespace=exposition-benchmark \
        --for=condition=Ready pod/nginx

    echo "Create the NamespaceOffloading resource and wait for namespace creation"
    $PROVIDER_EXEC -c 'cat <<EOF | kubectl apply -f -
apiVersion: offloading.liqo.io/v1alpha1
kind: NamespaceOffloading
metadata:
  name: offloading
  namespace: exposition-benchmark
spec:
  namespaceMappingStrategy: EnforceSameName
EOF'
    sleep 5 # Should be more than enough

    echo "Starting the measurer on the consumer cluster..."
    $CONSUMER_KUBECTL apply -f "/tmp/$EXPOSITION_MANIFEST_REMOTE_FILE"
    sleep 5 # Wait some time to ensure the pod starts

    echo "Starting the measurer on the provider cluster..."
    $PROVIDER_KUBECTL apply -f "/tmp/$EXPOSITION_MANIFEST_LOCAL_FILE"
    sleep 5 # Wait some time to ensure the pod starts

    echo "Waiting for the measurer on the provider cluster to complete..."
    while true; do
    TMP=$($PROVIDER_KUBECTL wait --timeout=-1s --namespace=exposition-syn-measurer \
        --for=condition=complete job/exposition-syn-measurer 2>&1 || true)
    if [[ "$TMP" == "job.batch/exposition-syn-measurer condition met" ]]; then break; fi
    done

    echo "Retrieving the resulting logs..."
    MEASURER=$($PROVIDER_KUBECTL get pod --namespace=exposition-syn-measurer -l app.kubernetes.io/name=exposition-syn-measurer \
        --output custom-columns=':.metadata.name' --no-headers)
    $PROVIDER_KUBECTL logs --namespace=exposition-syn-measurer "$MEASURER" > \
        "$OUTPUT/exposition-syn-vanilla-$RUN.txt"

    echo "Waiting for the measurer on the consumer cluster to complete..."
    while true; do
    TMP=$($CONSUMER_KUBECTL wait --timeout=-1s --namespace=exposition-syn-measurer \
        --for=condition=complete job/exposition-syn-measurer 2>&1 || true)
    if [[ "$TMP" == "job.batch/exposition-syn-measurer condition met" ]]; then break; fi
    done

    echo "Retrieving the resulting logs..."
    MEASURER=$($CONSUMER_KUBECTL get pod --namespace=exposition-syn-measurer -l app.kubernetes.io/name=exposition-syn-measurer \
        --output custom-columns=':.metadata.name' --no-headers)
    $CONSUMER_KUBECTL logs --namespace=exposition-syn-measurer "$MEASURER" > \
        "$OUTPUT/exposition-syn-liqo-$RUN.txt"

    echo "Resetting the environment..."
    $PROVIDER_KUBECTL delete namespace exposition-benchmark exposition-syn-measurer --ignore-not-found
    $CONSUMER_KUBECTL delete namespace exposition-benchmark exposition-syn-measurer --ignore-not-found
    while true; do
    TMP=$($PROVIDER_KUBECTL get pods -A -l app.kubernetes.io/part-of=benchmarks 2>&1)
    if [[ "$TMP" == "No resources found" ]]; then break; fi
    sleep 1
    done
    echo "Waiting a bit before staring the next benchmark..."
    sleep 15
done
