
# Image URL to use all building/pushing image targets
IMG ?= linkerd-config-controller:latest

# The name of the Kind cluster
KIND_CLUSTER ?= linkerd

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | linkerd inject --manual - | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.4 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

clean:
	kubectl -n linkerd delete all -l controllers.linkerd.io=linkerd-config
	kubectl -n linkerd delete role,rolebinding -l controllers.linkerd.io=linkerd-config
	kubectl delete clusterrole,clusterrolebinding  -l controllers.linkerd.io=linkerd-config
	kubectl delete mutatingwebhookconfiguration,validatingwebhookconfiguration -l controllers.linkerd.io=linkerd-config

purge: clean
	kubectl delete ns cert-manager
	linkerd install --ignore-cluster | kubectl delete -f -

cert-manager:
	kubectl create ns cert-manager
	kubectl label ns cert-manager config.linkerd.io/admission-webhooks=disabled
	kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v0.12.0/cert-manager.yaml

controller: docker-build kind-load deploy

emojivoto:
	linkerd inject https://run.linkerd.io/emojivoto.yml | kubectl apply -f -

kind-cluster:
	kind create cluster --name=${KIND_CLUSTER}

kind-load:
	kind load docker-image linkerd-config-controller --name=${KIND_CLUSTER}

.PHONY: linkerd
linkerd:
	kubectl label ns kube-system config.linkerd.io/admission-webhooks=disabled
	linkerd install | kubectl apply -f -
