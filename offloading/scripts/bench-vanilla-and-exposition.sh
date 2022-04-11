#!/bin/bash

set -e
set -o pipefail

if [ $# -ne 4 ]; then
    echo "Usage: $0 offloading-manifest exposition-manifest-local exposition-manifest-remote output-folder"
    echo "Measurer manifest not provided"
    exit 1
fi

OFFLOADING_MANIFEST=$1
OFFLOADING_MANIFEST_FILE=$(basename "$OFFLOADING_MANIFEST")
EXPOSITION_MANIFEST_LOCAL=$2
EXPOSITION_MANIFEST_LOCAL_FILE=$(basename "$EXPOSITION_MANIFEST_LOCAL")
EXPOSITION_MANIFEST_REMOTE=$3
EXPOSITION_MANIFEST_REMOTE_FILE=$(basename "$EXPOSITION_MANIFEST_REMOTE")
OUTPUT=$4

RUNS=10
DEPLOYS=1
PODS_ARRAY=(10 100 1000 10000)

NAMESPACE="liqo-benchmarks"

echo "Retrieving the configuration parameters..."
echo "Namespace: $NAMESPACE"
KUBECTL="kubectl --namespace $NAMESPACE"
CONSUMER=$($KUBECTL get pod -l app.kubernetes.io/component=consumer --output custom-columns=':.metadata.name' --no-headers)
PROVIDER=$($KUBECTL get pod -l app.kubernetes.io/component=provider --output custom-columns=':.metadata.name' --no-headers)
CONSUMER_EXEC="$KUBECTL exec $CONSUMER -c k3s-server -- /bin/sh"
PROVIDER_EXEC="$KUBECTL exec $PROVIDER -c k3s-server -- /bin/sh"
CONSUMER_KUBECTL="$KUBECTL exec $CONSUMER -c k3s-server -- kubectl"
PROVIDER_KUBECTL="$KUBECTL exec $PROVIDER -c k3s-server -- kubectl"


echo "Copying the measurer manifests to the provider..."
tar cf - -C "$(dirname $OFFLOADING_MANIFEST)" "$(basename $OFFLOADING_MANIFEST)" | \
    $KUBECTL exec "$PROVIDER" -c k3s-server -i -- tar xf - -C "/tmp"
tar cf - -C "$(dirname $EXPOSITION_MANIFEST_LOCAL)" "$(basename $EXPOSITION_MANIFEST_LOCAL)" | \
    $KUBECTL exec "$PROVIDER" -c k3s-server -i -- tar xf - -C "/tmp"
$PROVIDER_EXEC -c 'cat <<EOF > /tmp/converter
sed "s/__DEPLOYS__/\$2/" "\$1" | sed "s/__PODS__/\$3/" > "\$1-current"
EOF'

echo "Copying the measurer manifests to the consumer..."
tar cf - -C "$(dirname $EXPOSITION_MANIFEST_REMOTE)" "$(basename $EXPOSITION_MANIFEST_REMOTE)" | \
    $KUBECTL exec "$CONSUMER" -c k3s-server -i -- tar xf - -C "/tmp"
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
        $PROVIDER_EXEC /tmp/converter "/tmp/$OFFLOADING_MANIFEST_FILE" "$DEPLOYS" "$PODS"
        $PROVIDER_KUBECTL apply -f "/tmp/$OFFLOADING_MANIFEST_FILE-current"

        echo "Waiting for the measurer to complete..."
        while true; do
        TMP=$($PROVIDER_KUBECTL wait --timeout=-1s --namespace=offloading-measurer \
            --for=condition=complete job/offloading-measurer 2>&1 || true)
        if [[ "$TMP" == "job.batch/offloading-measurer condition met" ]]; then break; fi
        done

        echo "Retrieving the resulting logs..."
        MEASURER=$($PROVIDER_KUBECTL get pod --namespace=offloading-measurer -l app.kubernetes.io/name=offloading-measurer \
            --output custom-columns=':.metadata.name' --no-headers)
        $PROVIDER_KUBECTL logs --namespace=offloading-measurer "$MEASURER" > \
            "$OUTPUT/offloading-vanilla-$DEPLOYS-$PODS-$RUN.txt"

        echo "Create the NamespaceOffloading resource and wait for namespace creation"
        $PROVIDER_EXEC -c 'cat <<EOF | kubectl apply -f -
apiVersion: offloading.liqo.io/v1alpha1
kind: NamespaceOffloading
metadata:
  name: offloading
  namespace: offloading-benchmark
spec:
  namespaceMappingStrategy: EnforceSameName
EOF'
        sleep 5 # Should be more than enough

        echo "Starting the measurer on the consumer cluster..."
        $CONSUMER_EXEC /tmp/converter "/tmp/$EXPOSITION_MANIFEST_REMOTE_FILE" "$DEPLOYS" "$PODS"
        $CONSUMER_KUBECTL apply -f "/tmp/$EXPOSITION_MANIFEST_REMOTE_FILE-current"
        sleep 5 # Wait some time to ensure the pod starts

        echo "Starting the measurer on the provider cluster..."
        $PROVIDER_EXEC /tmp/converter "/tmp/$EXPOSITION_MANIFEST_LOCAL_FILE" "$DEPLOYS" "$PODS"
        $PROVIDER_KUBECTL apply -f "/tmp/$EXPOSITION_MANIFEST_LOCAL_FILE-current"
        sleep 5 # Wait some time to ensure the pod starts

        echo "Waiting for the measurer on the provider cluster to complete..."
        while true; do
        TMP=$($PROVIDER_KUBECTL wait --timeout=-1s --namespace=exposition-measurer \
            --for=condition=complete job/exposition-measurer 2>&1 || true)
        if [[ "$TMP" == "job.batch/exposition-measurer condition met" ]]; then break; fi
        done

        echo "Retrieving the resulting logs..."
        MEASURER=$($PROVIDER_KUBECTL get pod --namespace=exposition-measurer -l app.kubernetes.io/name=exposition-measurer \
            --output custom-columns=':.metadata.name' --no-headers)
        $PROVIDER_KUBECTL logs --namespace=exposition-measurer "$MEASURER" > \
            "$OUTPUT/exposition-vanilla-$DEPLOYS-$PODS-$RUN.txt"

        echo "Waiting for the measurer on the consumer cluster to complete..."
        while true; do
        TMP=$($CONSUMER_KUBECTL wait --timeout=-1s --namespace=exposition-measurer \
            --for=condition=complete job/exposition-measurer 2>&1 || true)
        if [[ "$TMP" == "job.batch/exposition-measurer condition met" ]]; then break; fi
        done

        echo "Retrieving the resulting logs..."
        MEASURER=$($CONSUMER_KUBECTL get pod --namespace=exposition-measurer -l app.kubernetes.io/name=exposition-measurer \
            --output custom-columns=':.metadata.name' --no-headers)
        $CONSUMER_KUBECTL logs --namespace=exposition-measurer "$MEASURER" > \
            "$OUTPUT/exposition-liqo-$DEPLOYS-$PODS-$RUN.txt"

        echo "Resetting the environment..."
        $CONSUMER_KUBECTL delete namespace offloading-measurer exposition-measurer offloading-benchmark --ignore-not-found
        $PROVIDER_KUBECTL delete namespace offloading-measurer exposition-measurer offloading-benchmark --ignore-not-found
        while true; do
        TMP=$($PROVIDER_KUBECTL get pods -A -l app.kubernetes.io/part-of=benchmarks 2>&1)
        if [[ "$TMP" == "No resources found" ]]; then break; fi
        sleep 1
        done
        $PROVIDER_KUBECTL delete pods -A -l app.kubernetes.io/component=virtual-kubelet
        echo "Waiting a bit before starting the next benchmark..."
        sleep 30
    done
done
