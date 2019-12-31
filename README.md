# Advanced StatefulSet

This is an Advanced StatefulSet CRD implementation based on official
StatefulSet in Kubernetes 1.17.0.

This is an experimental project.

## Features

In addition to official StatefulSet, it adds one feature:

- Scale in at an arbitrary position: https://github.com/kubernetes/kubernetes/issues/83224

## Development

### Verify

```
make verify
```

### Unit Tests

```
make test
```

### Integration Tests

```
make test-integration
```

### E2E

```
make e2e
```

## Test it out

### start a cluster

```
curl -Lo ./kind https://github.com/kubernetes-sigs/kind/releases/download/v0.6.1/kind-$(uname)-amd64
chmod +x ./kind
./kind create cluster --image kindest/node:v1.16.3 --config hack/kindconfig.v1.16.3.yaml --name advanced-statefulset
export KUBECONFIG=$(kind get kubeconfig-path --name advanced-statefulset)
```

### install CRD

```
kubectl apply -f deployment/crd.v1.yaml
```

### run advanced statefulset controller locally

Open a new terminal and run controller:

```
hack/local-up.sh
```

### deploy a statefulset

```
kubectl apply -f examples/statefulset.yaml
```

### scale out

Note that `--resource-version` is required for CRD objects.

```
RESOURCE_VERSION=$(kubectl get statefulsets.pingcap.com web -ojsonpath='{.metadata.resourceVersion}')
kubectl scale --resource-version=$RESOURCE_VERSION --replicas=4 statefulsets.pingcap.com web
```

### scale in

```
RESOURCE_VERSION=$(kubectl get statefulsets.pingcap.com web -ojsonpath='{.metadata.resourceVersion}')
kubectl scale --resource-version=$RESOURCE_VERSION --replicas=3 statefulsets.pingcap.com web
```

### scale in at arbitrary position

We should set `delete-slots` annotations and decrement `spec.replicas` at the
same time.

```
kubectl apply -f examples/scale-in-statefulset.yaml 
```
