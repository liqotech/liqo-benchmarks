#!/bin/bash

set -e
set -o pipefail

if [ $# -ne 1 ]; then
    echo "Usage: $0 output-folder"
    exit 1
fi

OUTPUT=$1

NAMESPACE="liqo-benchmarks"

echo "Retrieving the configuration parameters..."
echo "Namespace: $NAMESPACE"
KUBECTL="kubectl --namespace $NAMESPACE"

mkdir --parent "$OUTPUT"

echo "Retrieving the hub consumption..."
HUB=$($KUBECTL get pod -l app.kubernetes.io/component=hub --output custom-columns=':.metadata.name' --no-headers)
HUB_KUBECTL="$KUBECTL exec $HUB -c k3s-server -- kubectl"
MEASURER=$($HUB_KUBECTL get pod --namespace=consumption-measurer -l app.kubernetes.io/name=consumption-measurer \
    --output custom-columns=':.metadata.name' --no-headers)
$HUB_KUBECTL logs --namespace=consumption-measurer "$MEASURER" > "$OUTPUT/hub.csv"
$KUBECTL logs "$HUB" -c network-measurer | grep 'liqo_network' >> "$OUTPUT/hub.csv"

echo "Retrieving the minions consumption..."
IDX=0
MINIONS=$($KUBECTL get pod -l app.kubernetes.io/component=minion --output custom-columns=':.metadata.name' --no-headers)
for MINION in $MINIONS; do
    echo "Retrieving the $MINION minion consumption..."
    MINION_KUBECTL="$KUBECTL exec $MINION -c k3s-server -- kubectl"
    MEASURER=$($MINION_KUBECTL get pod --namespace=consumption-measurer -l app.kubernetes.io/name=consumption-measurer \
        --output custom-columns=':.metadata.name' --no-headers)
    $MINION_KUBECTL logs --namespace=consumption-measurer "$MEASURER" > "$OUTPUT/minion-$IDX.csv"
    IDX=$((IDX+1))
done
