#!/bin/bash

set -e
set -o pipefail

if [ $# -ne 3 ]; then
    echo "Usage: $0 type measurer-manifest output-folder"
    exit 1
fi

TYPE=$1
MANIFEST=$2
MANIFEST_FILE=$(basename "$MANIFEST")
OUTPUT=$3

RUNS=10
DEPLOYS=1
PODS_ARRAY=(10 100 1000 10000)
if [ "$TYPE" = "admiralty" ]; then
    PODS_ARRAY=(10 100 1000)
fi

NAMESPACE="liqo-benchmarks"

echo "Retrieving the configuration parameters..."
echo "Namespace: $NAMESPACE"
KUBECTL="kubectl --namespace $NAMESPACE"
CONSUMER=$($KUBECTL get pod -l app.kubernetes.io/component=consumer --output custom-columns=':.metadata.name' --no-headers)
CONSUMER_EXEC="$KUBECTL exec $CONSUMER -c k3s-server -- /bin/sh"
CONSUMER_KUBECTL="$KUBECTL exec $CONSUMER -c k3s-server -- kubectl"

echo "Copying the measurer manifest to the consumer..."
tar cf - -C "$(dirname $MANIFEST)" "$(basename $MANIFEST)" | $KUBECTL exec "$CONSUMER" -c k3s-server -i -- tar xf - -C "/tmp"
$CONSUMER_EXEC -c 'cat <<EOF > /tmp/converter
sed "s/__DEPLOYS__/\$2/" "\$1" | sed "s/__PODS__/\$3/" > "\$1-current"
EOF'

mkdir --parent "$OUTPUT"
echo "Ready to start executing the benchmarks"
for RUN in $(seq 1 $RUNS); do
    for PODS in "${PODS_ARRAY[@]}"; do
        echo
        echo "Run $RUN - Deployments $DEPLOYS - Pods $PODS"
        echo "Starting the measurer"
        $CONSUMER_EXEC /tmp/converter "/tmp/$MANIFEST_FILE" "$DEPLOYS" "$PODS"
        $CONSUMER_KUBECTL apply -f "/tmp/$MANIFEST_FILE-current"

        echo "Waiting for the measurer to complete..."
        while true; do
        TMP=$($CONSUMER_KUBECTL wait --timeout=-1s --namespace=offloading-measurer \
            --for=condition=complete job/offloading-measurer 2>&1 || true)
        if [[ "$TMP" == "job.batch/offloading-measurer condition met" ]]; then break; fi
        done

        echo "Retrieving the resulting logs..."
        MEASURER=$($CONSUMER_KUBECTL get pod --namespace=offloading-measurer -l app.kubernetes.io/name=offloading-measurer \
            --output custom-columns=':.metadata.name' --no-headers)
        $CONSUMER_KUBECTL logs --namespace=offloading-measurer "$MEASURER" > \
            "$OUTPUT/offloading-$TYPE-$DEPLOYS-$PODS-$RUN.txt"

        echo "Resetting the environment..."
        $CONSUMER_KUBECTL delete namespace offloading-measurer offloading-benchmark --ignore-not-found
        $CONSUMER_KUBECTL delete deployments -A -l app.kubernetes.io/part-of=benchmarks
        $CONSUMER_KUBECTL delete pods -A -l app.kubernetes.io/part-of=benchmarks >/dev/null
        while true; do
        TMP=$($CONSUMER_KUBECTL get pods -A -l app.kubernetes.io/part-of=benchmarks 2>&1)
        if [[ "$TMP" == "No resources found" ]]; then break; fi
        sleep 1
        done
        $CONSUMER_KUBECTL delete pods -A -l app.kubernetes.io/component=virtual-kubelet
        $CONSUMER_KUBECTL delete pods -A -l k8s-app=virtual-kubelet
        $CONSUMER_KUBECTL delete pods -A -l app.kubernetes.io/instance=admiralty
        echo "Waiting a bit before staring the next benchmark..."
        sleep 30
    done
done
