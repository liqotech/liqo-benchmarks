#!/bin/bash

set -e
set -o pipefail

if [ $# -ne 3 ]; then
    echo "Usage: $0 storage-manifest output-folder suffix"
    echo "Measurer parameters not provided"
    exit 1
fi

STORAGE_MANIFEST=$1
STORAGE_MANIFEST_FILE=$(basename "$STORAGE_MANIFEST")
OUTPUT=$2
SUFFIX=$3

RUNS=10
REPLICAS=5

NAMESPACE="liqo-benchmarks"

echo "Retrieving the configuration parameters..."
echo "Namespace: $NAMESPACE"
KUBECTL="kubectl --namespace $NAMESPACE"
CONSUMER=$($KUBECTL get pod -l app.kubernetes.io/component=consumer --output custom-columns=':.metadata.name' --no-headers)
CONSUMER_EXEC="$KUBECTL exec $CONSUMER -c k3s-server -- /bin/sh"
CONSUMER_KUBECTL="$KUBECTL exec $CONSUMER -c k3s-server -- kubectl"


echo "Copying the measurer manifests to the consumer..."
tar cf - -C "$(dirname $STORAGE_MANIFEST)" "$(basename $STORAGE_MANIFEST)" | \
    $KUBECTL exec "$CONSUMER" -c k3s-server -i -- tar xf - -C "/tmp"
$CONSUMER_EXEC -c 'cat <<EOF > /tmp/converter
sed "s/__REPLICAS__/\$2/" "\$1" > "\$1-current"
EOF'

mkdir --parent "$OUTPUT"
echo "Ready to start executing the benchmarks"
for RUN in $(seq 1 $RUNS); do
    echo
    echo "Run $RUN"

    echo "Starting the measurer on the consumer cluster..."
    $CONSUMER_EXEC /tmp/converter "/tmp/$STORAGE_MANIFEST_FILE" "$REPLICAS"
    $CONSUMER_KUBECTL apply -f "/tmp/$STORAGE_MANIFEST_FILE-current"

    echo "Waiting for the measurer on the consumer cluster to complete..."
    while true; do
    TMP=$($CONSUMER_KUBECTL wait --timeout=-1s --namespace=storage-measurer \
        --for=condition=complete job/storage-measurer 2>&1 || true)
    if [[ "$TMP" == "job.batch/storage-measurer condition met" ]]; then break; fi
    done

    echo "Retrieving the resulting logs..."
    $CONSUMER_KUBECTL logs --namespace=storage-measurer job/storage-measurer > \
        "$OUTPUT/storage-create-$SUFFIX-$REPLICAS-$RUN.txt"

    echo "Resetting the environment (keeping the PVCs)..."
    $CONSUMER_KUBECTL delete namespace storage-measurer --ignore-not-found
    $CONSUMER_KUBECTL delete statefulset -n storage-benchmark --all
    echo "Waiting a bit before starting the next step..."
    sleep 15

    echo "Starting the measurer on the consumer cluster..."
    $CONSUMER_EXEC /tmp/converter "/tmp/$STORAGE_MANIFEST_FILE" "$REPLICAS"
    $CONSUMER_KUBECTL apply -f "/tmp/$STORAGE_MANIFEST_FILE-current"

    echo "Waiting for the measurer on the consumer cluster to complete..."
    while true; do
    TMP=$($CONSUMER_KUBECTL wait --timeout=-1s --namespace=storage-measurer \
        --for=condition=complete job/storage-measurer 2>&1 || true)
    if [[ "$TMP" == "job.batch/storage-measurer condition met" ]]; then break; fi
    done

    echo "Retrieving the resulting logs..."
    $CONSUMER_KUBECTL logs --namespace=storage-measurer job/storage-measurer > \
        "$OUTPUT/storage-attach-$SUFFIX-$REPLICAS-$RUN.txt"

    echo "Resetting the environment..."
    $CONSUMER_KUBECTL delete namespace storage-benchmark storage-measurer --ignore-not-found
    echo "Waiting a bit before starting the next benchmark..."
    sleep 15
done
