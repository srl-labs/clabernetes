.DEFAULT_GOAL := help

ifeq (set-chart-versions,$(firstword $(MAKECMDGOALS)))
  # use the rest as arguments for "set-chart-versions" directive
  BUMP_CHART_VERSION_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(BUMP_CHART_VERSION_ARGS):;@:)
endif

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

fmt: ## Run formatters
	gofumpt -w .
	gci write --skip-generated .
	golines -w .

lint: fmt ## Run linters
	golangci-lint run
	helm lint --quiet charts/clabernetes
	helm lint --quiet charts/clicker

test: ## Run unit tests
	gotestsum --format testname --hide-summary=skipped -- -coverprofile=cover.out `go list ./... | grep -v e2e`

test-race: ## Run unit tests with race flag
	gotestsum --format testname --hide-summary=skipped -- -race -coverprofile=cover.out `go list ./... | grep -v e2e`

test-e2e: ## Run e2e tests
	gotestsum --format testname --hide-summary=skipped -- -race -coverprofile=cover.out ./e2e/...

cov:  ## Produce html coverage report; removes all the generated bits for sanity reasons
	cat cover.out | grep -v "/generated/" | grep -v "zz_generated.deepcopy.go" > cover.out.clean && rm cover.out && mv cover.out.clean cover.out
	go tool cover -html=cover.out

install-tools: ## Install lint/test tools
	go install mvdan.cc/gofumpt@latest
	go install github.com/daixiang0/gci@latest
	go install github.com/segmentio/golines@latest
	go install gotest.tools/gotestsum@latest

install-code-generators: ## Install latest code-generator tools
	go install k8s.io/code-generator/cmd/deepcopy-gen@latest
	go install k8s.io/kube-openapi/cmd/openapi-gen@latest
	go install k8s.io/code-generator/cmd/client-gen@latest
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

run-deepcopy-gen: ## Run deepcopy-gen
	deepcopy-gen \
	--go-header-file hack/boilerplate.go.txt \
	--output-file zz_generated.deepcopy.go \
	github.com/srl-labs/clabernetes/apis/...

run-openapi-gen: ## Run openapi-gen
	openapi-gen \
	--go-header-file hack/boilerplate.go.txt \
	--output-dir generated/openapi \
	--output-file openapi_generated.go \
	--output-pkg github.com/srl-labs/clabernetes/generated/openapi \
	github.com/srl-labs/clabernetes/apis/...

run-client-gen: ## Run client-gen
	client-gen \
	--go-header-file hack/boilerplate.go.txt \
	--input-base github.com/srl-labs/clabernetes \
	--input apis/v1alpha1 \
	--output-dir generated \
	--output-pkg github.com/srl-labs/clabernetes/generated \
	--clientset-name clientset

run-generate-crds: ## Run controller-gen for crds
	controller-gen crd paths=./apis/... output:crd:dir=./charts/clabernetes/crds/

run-generate: install-code-generators run-deepcopy-gen run-openapi-gen run-client-gen run-generate-crds fmt ## Run all code gen tasks
	cp charts/clabernetes/crds/*.yaml assets/crd/

delete-generated: ## Deletes all zz_*.go (generated) files, and crds
	find . -name "zz_*.go" -exec rm {} \;
	rm charts/clabernetes/crds/*.yaml || true
	rm assets/crd/*.yaml || true
	rm -rf generated/*

build-manager: ## Builds the clabernetes manager container; typically built via devspace, but this is a handy shortcut for one offs.
	docker build -t ghcr.io/srl-labs/clabernetes/clabernetes-manager:latest -f ./build/manager.Dockerfile .

build-launcher: ## Builds the clabernetes launcher container; typically built via devspace, but this is a handy shortcut for one offs.
	docker build -t ghcr.io/srl-labs/clabernetes/clabernetes-launcher:latest -f ./build/launcher.Dockerfile .

build-clabverter: ## Builds the clabverter container; typically built via devspace, but this is a handy shortcut for one offs.
	docker build -t ghcr.io/srl-labs/clabernetes/clabverter:latest -f ./build/clabverter.Dockerfile .

set-chart-versions: ## Sets the helm chart versions to the given value.
	./hack/set-chart-versions.sh $(BUMP_CHART_VERSION_ARGS)
