#!/bin/bash

set -e
set -o pipefail

NAMESPACE="liqo-benchmarks"

echo "Retrieving the configuration parameters..."
echo "Namespace: $NAMESPACE"

KUBECTL="kubectl --namespace $NAMESPACE"
CONSUMER=$($KUBECTL get pod -l app.kubernetes.io/component=consumer --output custom-columns=':.metadata.name' --no-headers)
PROVIDER=$($KUBECTL get pod -l app.kubernetes.io/component=provider --output custom-columns=':.metadata.name' --no-headers)
PROVIDERIP=$($KUBECTL get po -l app.kubernetes.io/component=provider -o custom-columns=':status.podIP' --no-headers)

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

echo "Creating the tensile kube namespace..."
$CONSUMER_EXEC 'cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Namespace
metadata:
  name: tensile
EOF'

echo "Retrieving the authentication info from the provider..."
PROVIDER_KUBECONF=$($PROVIDER_EXEC 'cat /etc/rancher/k3s/k3s.yaml | base64 --wrap=0')
$CONSUMER_EXEC "echo $PROVIDER_KUBECONF | base64 -d > /tmp/kubeconfig"
$CONSUMER_EXEC "sed -i \"s|127.0.0.1|$PROVIDERIP|\" /tmp/kubeconfig"
$CONSUMER_KUBECTL delete configmap kube --namespace=tensile --ignore-not-found
$CONSUMER_KUBECTL create configmap kube --namespace=tensile --from-file=/tmp/kubeconfig

echo "Deploying tensile kube..."
$CONSUMER_EXEC 'cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: virtual-kubelet
  namespace: tensile
  labels:
    k8s-app: virtual-kubelet
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: virtual-kubelet
  namespace: tensile
subjects:
- kind: ServiceAccount
  name: virtual-kubelet
  namespace: tensile
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: virtual-kubelet
  namespace: tensile
  labels:
    k8s-app: kubelet
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: virtual-kubelet
  template:
    metadata:
      labels:
        pod-type: virtual-kubelet
        k8s-app: virtual-kubelet
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: type
                    operator: NotIn
                    values:
                      - virtual-kubelet
      containers:
        - name: virtual-kubelet
          image: giorio94/virtual-node:v0.1.1-24-g2bd91c26b8c98c
          env:
            - name: KUBELET_PORT
              value: "10450"
            - name: VKUBELET_POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
          args:
            - --provider=k8s
            - --nodename=virtual-kubelet
            - --disable-taint=true
            - --kube-api-qps=500
            - --kube-api-burst=1000
            - --client-qps=500
            - --client-burst=1000
            - --client-kubeconfig=/root/kubeconfig
            - --klog.v=5
            - --log-level=debug
            - --metrics-addr=:10455
          volumeMounts:
          - name: kube
            mountPath: "/root"
            readOnly: true
          livenessProbe:
            tcpSocket:
              port: 10455
            initialDelaySeconds: 20
            periodSeconds: 20
      volumes:
        - name: kube
          configMap:
            name: kube
            items:
              - key: kubeconfig
                path: kubeconfig
            defaultMode: 420
      serviceAccountName: virtual-kubelet
EOF'

$CONSUMER_EXEC 'cat <<EOF | kubectl apply -f -
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: vk-mutator
  annotations:
    cert-manager.io/inject-ca-from: tensile/vk-mutator
webhooks:
- clientConfig:
    service:
      name: vk-mutator
      namespace: tensile
      path: /mutate
  failurePolicy: Ignore
  name: vk-mutator.tensile.svc
  rules:
    - apiGroups:
      - ""
      apiVersions:
      - v1
      operations:
      - CREATE
      resources:
      - pods
  sideEffects: None
---
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: self-signed
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: vk-mutator
  namespace: tensile
spec:
  secretName: vk-mutator
  dnsNames:
  - vk-mutator.tensile.svc
  issuerRef:
    kind: ClusterIssuer
    name: self-signed
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: vk-mutator
  name: vk-mutator
  namespace: tensile
spec:
  ports:
  - port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: vk-mutator
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: vk-mutator
  name: webhook
  namespace: tensile
spec:
  replicas: 1
  selector:
    matchLabels:
      app: vk-mutator
  template:
    metadata:
      labels:
        app: vk-mutator
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: type
                    operator: NotIn
                    values:
                      - virtual-kubelet
      containers:
      - args:
        - --tlscert=/root/tls.crt
        - --tlskey=/root/tls.key
        - --port=443
        - --v=6
        image: giorio94/virtual-webhook:v0.1.1-24-g2bd91c26b8c98c
        name: webhook
        ports:
        - containerPort: 443
          protocol: TCP
        volumeMounts:
        - mountPath: /root
          name: wbssecret
      volumes:
      - name: wbssecret
        secret:
          defaultMode: 420
          items:
            - key: tls.key
              path: tls.key
            - key: tls.crt
              path: tls.crt
          secretName: vk-mutator
EOF'

# Prevent offloaded pods from being scheduled on the k3s node
echo "Cordoning the provider node to force pods to be scheduled on hollow nodes..."
$PROVIDER_KUBECTL cordon "$PROVIDER"
$PROVIDER_KUBECTL taint node -l node-role.kubernetes.io/hollow=true node-role.kubernetes.io/hollow- || true

echo
echo "Provider: $KUBECTL exec $PROVIDER -it -- /bin/sh"
echo "Consumer: $KUBECTL exec $CONSUMER -it -- /bin/sh"
