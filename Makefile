.DEFAULT_GOAL := help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

fmt: ## Run formatters
	gofumpt -w .
	goimports -w .
	golines -w .

lint: fmt ## Run linters; runs with GOOS env var for linting on darwin
	golangci-lint run

test: ## Run unit tests
	gotestsum --format testname --hide-summary=skipped -- -coverprofile=cover.out ./...

test-race: ## Run unit tests with race flag
	gotestsum --format testname --hide-summary=skipped -- -race -coverprofile=cover.out ./...

cov:  ## Produce html coverage report
	go tool cover -html=cover.out

install-code-generators: ## Install latest code-generator tools
	go install k8s.io/code-generator/cmd/deepcopy-gen@latest
	go install k8s.io/code-generator/cmd/openapi-gen@latest
	go install k8s.io/code-generator/cmd/client-gen@latest
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

run-deepcopy-gen: ## Run deepcopy-gen
	GOMOD111=on \
	deepcopy-gen \
	--go-header-file hack/boilerplate.go.txt \
	--input-dirs github.com/srl-labs/clabernetes/apis/... \
	--output-file-base zz_generated.deepcopy \
	--trim-path-prefix ${GOPATH}/src/github.com/srl-labs/clabernetes

run-openapi-gen: ## Run openapi-gen
	GOMOD111=on \
	openapi-gen \
	--go-header-file hack/boilerplate.go.txt \
	--input-dirs github.com/srl-labs/clabernetes/apis/... \
	--trim-path-prefix ${GOPATH}/src/github.com/srl-labs/clabernetes \
	--output-package github.com/srl-labs/clabernetes/generated/openapi

run-client-gen: ## Run client-gen
	GOMOD111=on \
	client-gen \
	--go-header-file hack/boilerplate.go.txt \
	--input-base github.com/srl-labs/clabernetes/apis \
	--input topology/v1alpha1 \
	--trim-path-prefix ${GOPATH}/src/github.com/srl-labs/clabernetes \
	--output-package github.com/srl-labs/clabernetes/generated \
	--clientset-name clientset

run-generate-crds: ## Run controller-gen for crds
	controller-gen crd paths=./apis/... output:crd:dir=./chart/crds/

run-generate: install-code-generators run-deepcopy-gen run-openapi-gen run-client-gen run-generate-crds fmt ## Run all code gen tasks

delete-generated: ## Deletes all zz_*.go (generated) files, and crds
	find . -name "zz_*.go" -exec rm {} \;
	rm chart/crds/*.yaml || true
	rm -rf generated/*

refresh-chart-dependencies: ## Refreshes all dependent helm charts (ex: ui)
	cd chart; \
	helm dependency build; \
	helm dependency update

build-clabverter: ## Builds the clabverter container
	docker build -t ghcr.io/srl-labs/clabernetes/clabverter:latest -f ./build/clabverter.Dockerfile .