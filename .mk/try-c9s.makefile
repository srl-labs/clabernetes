TRY_C9S_CLUSTER_NAME ?= c9s-demo
TRY_C9S_CHART ?= oci://ghcr.io/srl-labs/clabernetes/clabernetes
TRY_C9S_CHART_VERSION ?=
TRY_C9S_TOPOLOGY ?= examples/basic/srl-multitool.yaml
TRY_C9S_TOPOLOGY_NAME ?= srl-multitool
TRY_C9S_UI_PORT ?= 3000
TRY_C9S_TIMEOUT ?= 600s

TRY_C9S_NAMESPACE := clabernetes
TRY_C9S_BUILD_DIR := build/try-c9s
TRY_C9S_STATE_DIR := $(TRY_C9S_BUILD_DIR)/$(TRY_C9S_CLUSTER_NAME)
TRY_C9S_TOOLS_DIR := $(TRY_C9S_BUILD_DIR)/bin
TRY_C9S_TMP_DIR := $(TRY_C9S_BUILD_DIR)/tmp
TRY_C9S_KIND_VERSION := v0.32.0
TRY_C9S_KUBECTL_VERSION := v1.36.1
TRY_C9S_HELM_VERSION := v4.2.0
TRY_C9S_OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
TRY_C9S_ARCH_QUERY := $(shell uname -m)
ifeq ($(TRY_C9S_ARCH_QUERY),x86_64)
TRY_C9S_ARCH := amd64
else ifeq ($(TRY_C9S_ARCH_QUERY),amd64)
TRY_C9S_ARCH := amd64
else ifeq ($(TRY_C9S_ARCH_QUERY),aarch64)
TRY_C9S_ARCH := arm64
else ifeq ($(TRY_C9S_ARCH_QUERY),arm64)
TRY_C9S_ARCH := arm64
else
TRY_C9S_ARCH := $(TRY_C9S_ARCH_QUERY)
endif
TRY_C9S_KIND := $(shell command -v kind 2>/dev/null || echo $(TRY_C9S_TOOLS_DIR)/kind)
TRY_C9S_KUBECTL := $(shell command -v kubectl 2>/dev/null || echo $(TRY_C9S_TOOLS_DIR)/kubectl)
TRY_C9S_HELM := $(shell command -v helm 2>/dev/null || echo $(TRY_C9S_TOOLS_DIR)/helm)
TRY_C9S_CHART_VERSION_ARG := $(if $(TRY_C9S_CHART_VERSION),--version $(TRY_C9S_CHART_VERSION),)
TRY_C9S_HELM_WAIT_ARG := $(shell if $(TRY_C9S_HELM) version --short 2>/dev/null | grep -q '^v4'; then echo '--wait=legacy'; else echo '--wait'; fi)

.PHONY: try-c9s
try-c9s: try-c9s-expose ## Launch published clabernetes in KinD and apply a sample topology
	@echo "--> TRY-C9S: clabernetes is ready to try"

.PHONY: try-c9s-tools
try-c9s-tools:
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "--> TRY-C9S: missing required tool: docker"; \
		exit 1; \
	fi
	@docker info >/dev/null 2>&1 || { echo "--> TRY-C9S: docker is not reachable"; exit 1; }
	@mkdir -p "$(TRY_C9S_TOOLS_DIR)" "$(TRY_C9S_TMP_DIR)"
	@if ! command -v "$(TRY_C9S_KIND)" >/dev/null 2>&1; then \
		if ! command -v curl >/dev/null 2>&1; then echo "--> TRY-C9S: curl is required to download kind"; exit 1; fi; \
		echo "--> TRY-C9S: downloading kind $(TRY_C9S_KIND_VERSION) to $(TRY_C9S_KIND)"; \
		curl -fsSL -o "$(TRY_C9S_KIND)" "https://kind.sigs.k8s.io/dl/$(TRY_C9S_KIND_VERSION)/kind-$(TRY_C9S_OS)-$(TRY_C9S_ARCH)"; \
		chmod +x "$(TRY_C9S_KIND)"; \
	fi
	@if ! command -v "$(TRY_C9S_KUBECTL)" >/dev/null 2>&1; then \
		if ! command -v curl >/dev/null 2>&1; then echo "--> TRY-C9S: curl is required to download kubectl"; exit 1; fi; \
		echo "--> TRY-C9S: downloading kubectl $(TRY_C9S_KUBECTL_VERSION) to $(TRY_C9S_KUBECTL)"; \
		curl -fsSL -o "$(TRY_C9S_KUBECTL)" "https://dl.k8s.io/release/$(TRY_C9S_KUBECTL_VERSION)/bin/$(TRY_C9S_OS)/$(TRY_C9S_ARCH)/kubectl"; \
		chmod +x "$(TRY_C9S_KUBECTL)"; \
	fi
	@if ! command -v "$(TRY_C9S_HELM)" >/dev/null 2>&1; then \
		if ! command -v curl >/dev/null 2>&1; then echo "--> TRY-C9S: curl is required to download helm"; exit 1; fi; \
		if ! command -v tar >/dev/null 2>&1; then echo "--> TRY-C9S: tar is required to install helm"; exit 1; fi; \
		echo "--> TRY-C9S: downloading helm $(TRY_C9S_HELM_VERSION) to $(TRY_C9S_HELM)"; \
		rm -rf "$(TRY_C9S_TMP_DIR)/helm-$(TRY_C9S_OS)-$(TRY_C9S_ARCH)"; \
		mkdir -p "$(TRY_C9S_TMP_DIR)/helm-$(TRY_C9S_OS)-$(TRY_C9S_ARCH)"; \
		curl -fsSL -o "$(TRY_C9S_TMP_DIR)/helm.tar.gz" "https://get.helm.sh/helm-$(TRY_C9S_HELM_VERSION)-$(TRY_C9S_OS)-$(TRY_C9S_ARCH).tar.gz"; \
		tar -xzf "$(TRY_C9S_TMP_DIR)/helm.tar.gz" -C "$(TRY_C9S_TMP_DIR)/helm-$(TRY_C9S_OS)-$(TRY_C9S_ARCH)"; \
		mv "$(TRY_C9S_TMP_DIR)/helm-$(TRY_C9S_OS)-$(TRY_C9S_ARCH)/$(TRY_C9S_OS)-$(TRY_C9S_ARCH)/helm" "$(TRY_C9S_HELM)"; \
		chmod +x "$(TRY_C9S_HELM)"; \
	fi

.PHONY: try-c9s-kind-config
try-c9s-kind-config: try-c9s-tools
	@mkdir -p "$(TRY_C9S_STATE_DIR)"
	@echo "--> TRY-C9S: writing KinD config $(TRY_C9S_STATE_DIR)/kind.yaml"
	@{ \
		printf '%s\n' '---'; \
		printf '%s\n' 'kind: Cluster'; \
		printf '%s\n' 'apiVersion: kind.x-k8s.io/v1alpha4'; \
		printf '%s\n' 'networking:'; \
		printf '%s\n' '  ipFamily: dual'; \
		printf '%s\n' 'nodes:'; \
		printf '%s\n' '  - role: control-plane'; \
		printf '%s\n' '    extraPortMappings:'; \
		printf '%s\n' '      - containerPort: 32767'; \
		printf '%s\n' '        hostPort: $(TRY_C9S_UI_PORT)'; \
		printf '%s\n' '        listenAddress: "0.0.0.0"'; \
		printf '%s\n' '    kubeadmConfigPatches:'; \
		printf '%s\n' '      - |'; \
		printf '%s\n' '        kind: KubeletConfiguration'; \
		printf '%s\n' '        streamingConnectionIdleTimeout: "96h0m0s"'; \
	} > "$(TRY_C9S_STATE_DIR)/kind.yaml"

.PHONY: try-c9s-cluster
try-c9s-cluster: try-c9s-kind-config
	@echo "--> TRY-C9S: creating KinD cluster $(TRY_C9S_CLUSTER_NAME)"
	@clusters=$$($(TRY_C9S_KIND) get clusters 2>/dev/null | grep -v '^No kind clusters found\.$$' || true); \
	if [ -n "$$clusters" ]; then \
		echo "--> TRY-C9S: found existing KinD cluster(s):"; \
		echo "$$clusters" | sed 's/^/    /'; \
		echo "--> TRY-C9S: run 'make try-c9s-clean' before starting try-c9s"; \
		exit 1; \
	fi
	@$(TRY_C9S_KIND) create cluster --name $(TRY_C9S_CLUSTER_NAME) --config "$(TRY_C9S_STATE_DIR)/kind.yaml"
	@$(TRY_C9S_KIND) export kubeconfig --name $(TRY_C9S_CLUSTER_NAME)
	@$(TRY_C9S_KUBECTL) wait --for=condition=Ready nodes --all --timeout=$(TRY_C9S_TIMEOUT)

.PHONY: try-c9s-metallb
try-c9s-metallb: try-c9s-cluster
	@echo "--> TRY-C9S: installing MetalLB"
	@$(TRY_C9S_KUBECTL) apply -f "https://raw.githubusercontent.com/metallb/metallb/v0.15.3/config/manifests/metallb-native.yaml"
	@$(TRY_C9S_KUBECTL) -n metallb-system wait --for=condition=Ready pods --selector=app=metallb --timeout=120s
	@echo "--> TRY-C9S: configuring MetalLB address pool from Docker network kind"
	@ipv4_subnet=$$(docker network inspect -f '{{range .IPAM.Config}}{{.Subnet}} {{end}}' kind | tr ' ' '\n' | grep -v ':' | head -n 1); \
	ipv6_subnet=$$(docker network inspect -f '{{range .IPAM.Config}}{{.Subnet}} {{end}}' kind | tr ' ' '\n' | grep ':' | head -n 1); \
	if [ -z "$$ipv4_subnet" ]; then echo "--> TRY-C9S: could not detect IPv4 subnet for Docker network kind"; exit 1; fi; \
	ipv4_prefix=$$(echo "$$ipv4_subnet" | awk -F. '{print $$1 "." $$2}'); \
	ipv4_pool="$${ipv4_prefix}.255.0/24"; \
	ipv6_pool=""; \
	if [ -n "$$ipv6_subnet" ]; then \
		ipv6_prefix=$$(echo "$$ipv6_subnet" | awk -F: '{print $$1 ":" $$2 ":" $$3 ":" $$4}'); \
		ipv6_pool="$${ipv6_prefix}:ffff:ffff:ffff:ffff/120"; \
	fi; \
	{ \
		printf '%s\n' '---'; \
		printf '%s\n' 'apiVersion: metallb.io/v1beta1'; \
		printf '%s\n' 'kind: IPAddressPool'; \
		printf '%s\n' 'metadata:'; \
		printf '%s\n' '  name: kind'; \
		printf '%s\n' '  namespace: metallb-system'; \
		printf '%s\n' 'spec:'; \
		printf '%s\n' '  addresses:'; \
		printf '  - %s\n' "$$ipv4_pool"; \
		if [ -n "$$ipv6_pool" ]; then printf '  - %s\n' "$$ipv6_pool"; fi; \
		printf '%s\n' '  avoidBuggyIPs: true'; \
		printf '%s\n' '---'; \
		printf '%s\n' 'apiVersion: metallb.io/v1beta1'; \
		printf '%s\n' 'kind: L2Advertisement'; \
		printf '%s\n' 'metadata:'; \
		printf '%s\n' '  name: kind'; \
		printf '%s\n' '  namespace: metallb-system'; \
		printf '%s\n' 'spec:'; \
		printf '%s\n' '  ipAddressPools:'; \
		printf '%s\n' '    - kind'; \
	} | $(TRY_C9S_KUBECTL) apply -f -

.PHONY: try-c9s-install
try-c9s-install: try-c9s-metallb
	@echo "--> TRY-C9S: installing published clabernetes chart"
	@$(TRY_C9S_HELM) upgrade --install clabernetes $(TRY_C9S_CHART) $(TRY_C9S_CHART_VERSION_ARG) \
		--namespace $(TRY_C9S_NAMESPACE) \
		--create-namespace \
		$(TRY_C9S_HELM_WAIT_ARG) \
		--timeout $(TRY_C9S_TIMEOUT) \
		--set ui.ingress.enabled=false \
		--set manager.replicaCount=1 \
		--set ui.replicaCount=1
	@$(TRY_C9S_KUBECTL) -n $(TRY_C9S_NAMESPACE) rollout status deploy/clabernetes-manager --timeout=$(TRY_C9S_TIMEOUT)
	@$(TRY_C9S_KUBECTL) -n $(TRY_C9S_NAMESPACE) rollout status deploy/clabernetes-ui --timeout=$(TRY_C9S_TIMEOUT)

.PHONY: try-c9s-apply-topology
try-c9s-apply-topology: try-c9s-install
	@echo "--> TRY-C9S: applying sample topology $(TRY_C9S_TOPOLOGY)"
	@$(TRY_C9S_KUBECTL) -n default apply -f $(TRY_C9S_TOPOLOGY)
	@echo "--> TRY-C9S: waiting up to $(TRY_C9S_TIMEOUT) for topology $(TRY_C9S_TOPOLOGY_NAME)"
	@if ! $(TRY_C9S_KUBECTL) -n default wait \
		--for=condition=TopologyReady \
		topology/$(TRY_C9S_TOPOLOGY_NAME) \
		--timeout=$(TRY_C9S_TIMEOUT); then \
		echo "--> TRY-C9S: topology did not report ready before timeout; current status:"; \
		$(TRY_C9S_KUBECTL) -n default get topology $(TRY_C9S_TOPOLOGY_NAME) || true; \
		$(TRY_C9S_KUBECTL) -n default get pods -l clabernetes/topologyOwner=$(TRY_C9S_TOPOLOGY_NAME) || true; \
	fi

.PHONY: try-c9s-ui-service
try-c9s-ui-service: try-c9s-apply-topology
	@echo "--> TRY-C9S: creating fixed UI NodePort service"
	@mkdir -p "$(TRY_C9S_STATE_DIR)"
	@$(TRY_C9S_KUBECTL) -n default delete service try-c9s-srl --ignore-not-found=true >/dev/null 2>&1 || true
	@{ \
		printf '%s\n' '---'; \
		printf '%s\n' 'apiVersion: v1'; \
		printf '%s\n' 'kind: Service'; \
		printf '%s\n' 'metadata:'; \
		printf '%s\n' '  labels:'; \
		printf '%s\n' '    try-c9s: "true"'; \
		printf '%s\n' '  name: try-c9s-ui'; \
		printf '%s\n' '  namespace: $(TRY_C9S_NAMESPACE)'; \
		printf '%s\n' 'spec:'; \
		printf '%s\n' '  type: NodePort'; \
		printf '%s\n' '  ports:'; \
		printf '%s\n' '    - name: http'; \
		printf '%s\n' '      nodePort: 32767'; \
		printf '%s\n' '      port: 80'; \
		printf '%s\n' '      protocol: TCP'; \
		printf '%s\n' '      targetPort: 3000'; \
		printf '%s\n' '  selector:'; \
		printf '%s\n' '    clabernetes/app: clabernetes'; \
		printf '%s\n' '    clabernetes/name: clabernetes-ui'; \
		printf '%s\n' '    clabernetes/component: ui'; \
	} > "$(TRY_C9S_STATE_DIR)/ui-service.yaml"
	@$(TRY_C9S_KUBECTL) apply -f "$(TRY_C9S_STATE_DIR)/ui-service.yaml"

.PHONY: try-c9s-print-access
try-c9s-print-access:
	@srl_ip=$$($(TRY_C9S_KUBECTL) -n default get svc "$(TRY_C9S_TOPOLOGY_NAME)-srl1" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || true); \
	echo "--> TRY-C9S: UI: http://localhost:$(TRY_C9S_UI_PORT)"; \
	if [ -n "$$srl_ip" ]; then \
		echo "--> TRY-C9S: SR Linux SSH: ssh admin@$$srl_ip"; \
		echo "--> TRY-C9S: SR Linux gNMI: $$srl_ip:57400"; \
		echo "--> TRY-C9S: SR Linux NETCONF: $$srl_ip:830"; \
	else \
		echo "--> TRY-C9S: SR Linux service: kubectl -n default get svc $(TRY_C9S_TOPOLOGY_NAME)-srl1"; \
	fi

.PHONY: try-c9s-expose
try-c9s-expose: try-c9s-ui-service
	@if ! docker port "$(TRY_C9S_CLUSTER_NAME)-control-plane" 32767/tcp >/dev/null 2>&1; then \
		echo "--> TRY-C9S: KinD UI host port mapping is missing"; \
		echo "--> TRY-C9S: run 'make try-c9s-clean' and then 'make try-c9s'"; \
		exit 1; \
	fi
	@$(MAKE) --no-print-directory try-c9s-print-access

.PHONY: try-c9s-clean
try-c9s-clean: ## Remove try-c9s sample resources and KinD cluster
	@if command -v "$(TRY_C9S_KIND)" >/dev/null 2>&1 && $(TRY_C9S_KIND) get clusters | grep -qx '$(TRY_C9S_CLUSTER_NAME)'; then \
		$(TRY_C9S_KIND) export kubeconfig --name $(TRY_C9S_CLUSTER_NAME) >/dev/null; \
		$(TRY_C9S_KUBECTL) -n default delete -f $(TRY_C9S_TOPOLOGY) --ignore-not-found=true >/dev/null 2>&1 || true; \
		$(TRY_C9S_KIND) delete cluster --name $(TRY_C9S_CLUSTER_NAME); \
	else \
		echo "--> TRY-C9S: KinD cluster $(TRY_C9S_CLUSTER_NAME) does not exist"; \
	fi
