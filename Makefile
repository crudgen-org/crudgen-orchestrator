IMG         ?= mohik/crudgen-orchestrator
TAG         ?= latest
CRD_OPTIONS ?= "crd:trivialVersions=true"

ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

manager: generate fmt vet
	go build -o bin/manager main.go

run: generate fmt vet manifests
	go run ./main.go --root-domain aaas.crudgen.org --cluster-issuer lets-encrypt

install: manifests
	kustomize build config/crd | kubectl apply -f -

uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}:${TAG}
	kustomize build config/default | kubectl apply -f -

manifests: controller-gen
	$(CONTROLLER_GEN) \
		paths="./api/..." \
		crd:crdVersions=v1 \
		output:crd:dir=config/crd/bases

fmt:
	go fmt ./...

vet:
	go vet ./...

generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

docker-build: test
	docker build . -t ${IMG}:${TAG}

docker-push:
	docker push ${IMG}:${TAG}

docker: docker-build docker-push

controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
