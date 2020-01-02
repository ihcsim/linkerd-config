# linkerd-config
This experimental project introduces a new Kubernetes controller, named
`LinkerdConfig`, that knows how to reconcile the Linkerd's configuration.

The `LinkerdConfig` controller can be used to automatically:

* propagate new configuration to annotated proxies in the data plane
* revert any manual unsolicited changes made to the `linkerd-config` configmap

This project is tested with the following software:

* [Linkerd edge-19.12.1](https://linkerd.io)
* [KubeBuilder 2.2.0](https://kubebuilder.io/)
* [Kind v0.5.1](https://github.com/kubernetes-sigs/kind)
* [cert-manager v0.12.0](https://cert-manager.io/)

## Introduction

## Getting Started
In this scenario, we will auto-upgrade the proxy version of all opt-in pods
in the data plane with the following steps:

1. Install Linkerd edge-19.12.1
1. Deploy the `LinkerdConfig` controller
1. Install the `edge-19.12.3` `LinkerdConfig` custom resource
1. Let the `LinkerdConfig` controller reconcile the `linkerd-config` map with
the `edge-19.12.3` custom resource
1. Let the `LinkerdConfig` controller restart all opt-in emojivoto pods

Set up a Kind cluster, named `linkerd`:
```
make kind-cluster
```
(The name of the cluster can be overriden using the `KIND_CLUSTER` variable.)

Install cert-manager to manage the CA bundle of the controller's webhooks:
```
make cert-manager
```
(This is optional if you already have your own cert-manager.)

Install Linkerd:
```
linkerd install | kubectl apply -f -

linkerd check

linkerd version
Client version: edge-19.12.1
Server version: edge-19.12.1
```

Label the `kube-system` namespace so that the `LinkerdConfig` controller will
ignore all the system pods during reconciliation:
```
kubectl label ns kube-system config.linkerd.io/admission-webhooks=disabled
```

Deploy the `linkerdconfigs` custom resource definition:
```
make install

kubectl get crd linkerdconfigs.config.linkerd.io
NAME                               CREATED AT
linkerdconfigs.config.linkerd.io   2020-01-02T03:48:03Z
```

Build and deploy the `LinkerdConfig` controller:
```
make controller

kubectl -n linkerd get po linkerd-config-controller-manager-5b54566647-cqz9h
NAME                                                 READY   STATUS    RESTARTS   AGE
linkerd-config-controller-manager-5b54566647-cqz9h   3/3     Running   0          91s
```

Install and inject the emojivoto application:
```
make emojivoto

# confirm the proxy version is at edge-19.12.1.
# this will be auto-upgraded later.
kubectl -n emojivoto get po emoji-774d574dff-znjgp -ojsonpath="{.spec.containers[1].image}"
gcr.io/linkerd-io/proxy:edge-19.12.1

kubectl -n emojivoto get po voting-666f88b64b-cmsmd -ojsonpath="{.spec.containers[1].image}"
gcr.io/linkerd-io/proxy:edge-19.12.1
```
Note that the `emoji` and `voting` `Deployment`s' pod templates are labeled with
the `config.linkerd.io/reconcile: auto` label.

Install the `edge_19.12.3` custom resource:
```
kubectl apply -f config/samples/edge-19.12.3.yaml
```

Notice that the `linkerd-config` configmap's data has been updated to match the
defaults defined in the `edge-19.12.3` custom resource:
```
kubectl -n linkerd describe cm linkerd-config |less
...
Data
====
global:
----
{"linkerdNamespace":"linkerd","cniEnabled":false,"version":"edge-19.12.3",...
```
In addition, it also has an `ownerReference` pointing to the custom resource:
```
kubectl -n linkerd get cm linkerd-config -oyaml | less
...
  ownerReferences:
  - apiVersion: config.linkerd.io/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: LinkerdConfig
    name: edge-19.12.3
    uid: 89235148-79e0-4120-b10a-c682f3b96db9
```

Take a look at your emojivoto application. Both the `emoji` and `voting` pods
should have auto-restarted:
```
kubectl -n emojivoto get po
NAME                       READY   STATUS    RESTARTS   AGE
emoji-9bf75cd-xgtnv         2/2     Running   0          58s
vote-bot-6fbc5f65f8-q2brn   2/2     Running   0          16m
voting-5c545ff9cd-8gzvh     2/2     Running   0          58s
web-744588c478-gmwc2        2/2     Running   0          16m

# confirm that the proxy version is updated to edge-19.12.3
kubectl -n emojivoto get po emoji-9bf75cd-xgtnv -ojsonpath="{.spec.containers[1].image}"
gcr.io/linkerd-io/proxy:edge-19.12.3

kubectl -n emojivoto get po voting-5c545ff9cd-8gzvh -ojsonpath="{.spec.containers[1].image}"
gcr.io/linkerd-io/proxy:edge-19.12.3
```

## Implementation Highlights
The following are some implementation highlights:

* The controller watches the `linkerd-config` configmap. It makes making the `edge-19.12.3` custom resource the owner of this configmap
* When the configmap is updated, the controller:
  1. Overrides all mismatching or additional data in the configmap, with the defaults defined in the `edge-19.12.3` custom resource
  1. Restart all injected pods that are labeled with the `config.linkerd.io/reconcile: auto` label. Note that these pods must not reside in namespaces which have the `config.linkerd.io/admission-webhooks: disabled` label
* The controller sets up a [`FieldIndexer`](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/client#FieldIndexer) on the pods' `Phase` field so that the [`Client`](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/client#Client) can query for `Running` pods
* [Predicate](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/predicate#ResourceVersionChangedPredicate) is used to respond to only "resource version changed" events of the custom resources
* Important events are published to the K8s event bus using the [`Recorder`](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/recorder)
* A mutating webhook is used to provide defaults to required fields in the custom resource
* A validating webhook is used to validate required user-managed fields (e.g., mTLS trust anchors)
* The controller and all its namespaced resources are installed in the `linkerd` namespace
* The controller is injected with the Linkerd proxy

## Development
To deploy the CRDs to K8s:
```
make install
```

To run the controller in standalone mode, outside of K8s:
```
make run ENABLE_WEBHOOKS=false
```

To build the controller Docker image, and load it into a Kind cluster:
```
make controller
```

To remove the controller's `Deployment` and other resources (e.g. RBAC):
```
make clean
```

## Future Work

The following is a list of future work:

* Since the `linkerd-config` configmap has an `ownerReference` pointing to the `edge-19.12.3` custom resource, the deletion of the custom resource will trigger a cascading delete on the configmap. The configmap's lifecycle should be managed independently to avoid breaking the control plane
* The reconciliation process with multiple `linkerdconfigs` custom resources is yet to be determined. We can either use the most-recent custom resource, or one that is labeled as `active`, or perform some form of 3-way merges among all the resources
* Re-use the data structure defined in the Linkerd `config.proto` file in the custom resource definition. Currently, the controller have to convert the custom resource definition into the protobuf format that Linkerd can consume
