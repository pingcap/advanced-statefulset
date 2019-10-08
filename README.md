# Advanced StatefulSet

[![Build Status](https://travis-ci.com/cofyc/advanced-statefulset.svg?token=d2ZWjsd7VLnLR8jqd2ay&branch=master)](https://travis-ci.com/cofyc/advanced-statefulset)
[![Docker Repository on Quay](https://quay.io/repository/cofyc/advanced-statefulset/status "Docker Repository on Quay")](https://quay.io/repository/cofyc/advanced-statefulset)

This is an Advanced StatefulSet CRD implementation based on official
StatefulSet in Kubernetes 1.16.0.

This is an experimental project.

## Features

In addition to official StatefulSet, it adds one feature:

- Scale in at an arbitrary position: https://github.com/kubernetes/kubernetes/issues/83224

## Test it out

### start a cluster

kind `v0.5.1` is required.

```
curl -Lo ./kind https://github.com/kubernetes-sigs/kind/releases/download/v0.5.1/kind-$(uname)-amd64
chmod +x ./kind
./kind create cluster --image kindest/node:v1.16.1 --config hack/kindconfig.yaml
export KUBECONFIG=$(kind get kubeconfig-path --name kind)
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

### scale up

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
