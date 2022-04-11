#!/bin/bash

set -e
set -o pipefail

if [ $# -ne 2 ]; then
    echo "Usage: $0 helm-repo output-folder"
    exit 1
fi

REPO=$1
OUTPUT=$2

RUNS=10
MINIONS_ARRAY=(1 2 4 8 16 32 64 128)
NAMESPACE="liqo-benchmarks"

echo "Retrieving the configuration parameters..."
echo "Namespace: $NAMESPACE"
KUBECTL="kubectl --namespace $NAMESPACE"
HELM="helm --namespace $NAMESPACE"

mkdir --parent "$OUTPUT"
echo "Ready to start executing the benchmarks"
for RUN in $(seq 1 $RUNS); do
    for MINIONS in "${MINIONS_ARRAY[@]}"; do
        echo
        echo "Run $RUN - Minions $MINIONS"

        echo "Starting the testbed"
        $HELM install liqo-k3s-cattle "$REPO" --set minion.replicaCount="$MINIONS" --wait --timeout=1h

        HUB=$($KUBECTL get pod -l app.kubernetes.io/name=liqo-k3s-cattle,app.kubernetes.io/component=hub \
            --output custom-columns=':.metadata.name' --no-headers)
        HUB_KUBECTL="$KUBECTL exec $HUB -c k3s-server -- kubectl"

        echo "Waiting for the measurer to complete..."
        while true; do
        TMP=$($HUB_KUBECTL wait --timeout=-1s --namespace=peering-measurer \
            --for=condition=complete job/peering-measurer 2>&1 || true)
        if [[ "$TMP" == "job.batch/peering-measurer condition met" ]]; then break; fi
        done

        echo "Retrieving the resulting logs..."
        MEASURER=$($HUB_KUBECTL get pod --namespace=peering-measurer -l app.kubernetes.io/name=peering-measurer \
            --output custom-columns=':.metadata.name' --no-headers)
        FILENAME=$(printf 'peering-%03d-%d.txt' "$MINIONS" "$RUN")
        $HUB_KUBECTL logs --namespace=peering-measurer "$MEASURER" > "$OUTPUT/$FILENAME"

        echo "Resetting the environment..."
        $HELM uninstall liqo-k3s-cattle

        while true; do
        TMP=$($KUBECTL get pods -A -l app.kubernetes.io/name=liqo-k3s-cattle 2>&1)
        if [[ "$TMP" == "No resources found" ]]; then break; fi
        sleep 1
        done

        echo "Waiting a bit before starting the next benchmark..."
        sleep 10
    done
done
