
# Image URL to use all building/pushing image targets
IMG ?= gcr.io/mercari-us-double/merlin:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Sample configs
SAMPLE_DIR := config/samples
SAMPLE_COFIGS := $(shell ls -d $(SAMPLE_DIR))

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[\/a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: generate imports vet manifests ## Run tests
	go test ./... -coverprofile coverage.out

manager: generate imports vet ## Build manager binary
	go build -o bin/manager main.go

run: generate imports vet manifests ## Run against the configured Kubernetes cluster in ~/.kube/config
	go run ./main.go

show-crd: manifests ## Show CRDs
	kustomize build config/crd

show-deploy: manifests ## Show Deploy configs
	kustomize build config/default

install: manifests ## Install CRDs into a cluster
	kustomize build config/crd | kubectl apply -f -

uninstall: manifests ## Uninstall CRDs from a cluster
	kustomize build config/crd | kubectl delete -f -

apply-samples: ## Apply samples
	@for SAMPLE in $(SAMPLE_COFIGS); do \
		kubectl apply -f $${SAMPLE}; \
	done

deploy: manifests ## Deploy controller in the configured Kubernetes cluster in ~/.kube/config
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

manifests: controller-gen ## Generate manifests e.g. CRD, RBAC etc.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

imports: ## Run go imports against code
ifeq (, $(shell which goimports))
	go get golang.org/x/tools/cmd/goimports
endif
	@for FILENAME in $$(find . -type f -name '*.go' -not -path "./vendor/*"); do \
		goimports -w $$FILENAME; \
	done

vet: ## Run go vet against code
	go vet ./...

generate: controller-gen ## Generate code
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

docker-build: test ## Build the docker image
	docker build . -t ${IMG}

docker-push: ## Push the docker image
	docker push ${IMG}

vendor: ## Getting libraries to vendor folder
	go mod vendor

mockgen: vendor ## Gen mocks - currently only k8s cilent mock
	mockgen -package mocks -source vendor/sigs.k8s.io/controller-runtime/pkg/client/interfaces.go -destination mocks/k8s_client_mock.go

controller-gen: ## find controller-gen, download controller-gen if necessary
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.9 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
