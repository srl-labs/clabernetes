[![Discord](https://img.shields.io/discord/860500297297821756?style=flat-square&label=discord&logo=discord&color=00c9ff&labelColor=bec8d2)](https://discord.gg/vAyddtaEV9)
[![Go Report](https://img.shields.io/badge/go%20report-A%2B-blue?style=flat-square&color=00c9ff&labelColor=bec8d2)](https://goreportcard.com/report/github.com/srl-labs/clabernetes)

# clabernetes aka c9s

Love containerlab? Want containerlab, just distributed in a kubernetes cluster? Enter
clabernetes -- containerlab + kubernetes. clabernetes is a kubernetes controller that deploys valid
containerlab topologies into a kubernetes cluster.

## Requirements

- A kubernetes cluster of course!
  - Requires `MixedProtocolLBService` feature gate (stable in 1.26, enabled in 1.24).
  - Local testing can be done with [kind](https://kind.sigs.k8s.io/).
- [Helm](https://helm.sh/docs/intro/install/) to install the clabernetes CRDs.
- [jq](https://jqlang.github.io/jq/download/) to support kube-vip load balancer installation.
- The ability to run privileged pods in your cluster.
- A load balancer in your cluster if you want to expose nodes "directly".
- Sufficient privileges to install the controller.

## Installation

```bash
helm upgrade --install clabernetes oci://ghcr.io/srl-labs/clabernetes/clabernetes
```

## Quickstart

This quickstart uses [kind](https://kind.sigs.k8s.io/) to create a local kubernetes cluster and
then deploys clabernetes into. If you already have a kubernetes cluster, you can skip the first
step.

### Creating a local multi-node kubernetes cluster with kind

Clabernetes goal is to allow users to run networking labs with the easy of use of containerlab,
but with the scaling powers of kubernetes. To simulate the scaling aspect, we'll use kind to
create a multi-node kubernetes cluster.

```bash
# creates a 3-node kubernetes cluster
kind create cluster --config examples/kind-cluster.yml
```

### Installing clabernetes

Clabernetes is installed into a kubernetes cluster using [helm](https://helm.sh/docs/intro/install/):

```bash
helm upgrade --install clabernetes oci://ghcr.io/srl-labs/clabernetes/clabernetes
```

A successful installation will result in a clabernetes-manager deployment of three pods running in
the cluster:

```bash
$ kubectl get pods -o wide
NAME                                   READY   STATUS    RESTARTS   AGE    IP           NODE           NOMINATED NODE   READINESS GATES
clabernetes-manager-85cf4ddbb5-7m8wt   1/1     Running   0          5m2s   10.244.2.3   kind-worker    <none>           <none>
clabernetes-manager-85cf4ddbb5-grpbh   1/1     Running   0          5m2s   10.244.1.2   kind-worker2   <none>           <none>
clabernetes-manager-85cf4ddbb5-k9rkk   1/1     Running   0          5m2s   10.244.2.2   kind-worker    <none>           <none>
```

### Installing Load Balancer

To get external access to the nodes deployed by clabernetes we will install kube-vip load
balancer into the cluster.

Following [kube-vip + kind](https://kube-vip.io/docs/usage/kind/) installation instructions:

```bash
kubectl apply -f https://kube-vip.io/manifests/rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/kube-vip/kube-vip-cloud-provider/main/manifest/kube-vip-cloud-controller.yaml
kubectl create configmap --namespace kube-system kubevip --from-literal range-global=172.18.1.10-172.18.1.250
```

Next we setup kube-vip's container image:

```bash
KVVERSION=$(curl -sL https://api.github.com/repos/kube-vip/kube-vip/releases | jq -r ".[0].name")
alias kube-vip="docker run --network host --rm ghcr.io/kube-vip/kube-vip:$KVVERSION"
```

And install kube-vip load balancer daemonset in ARP mode:

```bash
kube-vip manifest daemonset --services --inCluster --arp --interface eth0 | kubectl apply -f -
```

We can check kuve-vip daemonset pods are running on both worker nodes:

```bash
❯ k get pods -A -o wide | grep kube-vip
kube-system          kube-vip-cloud-provider-54c878b6c5-tlvh5     1/1     Running   0          2m34s   10.244.0.5   kind-control-plane   <none>           <none>
kube-system          kube-vip-ds-jn8qg                            1/1     Running   0          84s     172.18.0.3   kind-worker2         <none>           <none>
kube-system          kube-vip-ds-tmfrq                            1/1     Running   0          84s     172.18.0.4   kind-worker          <none>           <none>
```

### Deploying a topology

Clabernetes uses the same topology format as containerlab. Take a look at the simple
[2-node topology](examples/two-srl.c9s.yml) consisting of two SR Linux nodes:

```yaml
---
apiVersion: topology.clabernetes/v1alpha1
kind: Containerlab
metadata:
  name: clab-srl02
  namespace: srl02
spec:
  # nodes' startup config is omitted for brevity
  config: |-
    name: srl02
    topology:
      nodes:
        srl1:
          kind: srl
          image: ghcr.io/nokia/srlinux:23.7.1

        srl2:
          kind: srl
          image: ghcr.io/nokia/srlinux:23.7.1
      links:
        - endpoints: ["srl1:e1-1", "srl2:e1-1"]
```

As you can see, the familiar Containerlab topology is wrapped in a `Containerlab` Custom
Resource. The `spec.config` field contains the topology definition. The `metadata.name` field is
the name of the topology. The `metadata.namespace` field is the namespace in which the topology
will be deployed.

Before deploying this lab we need to create the namespace as set in our Clabernetes resource:

```bash
kubectl create namespace srl02
```

And now we are ready to deploy our first clabernetes topology:

```bash
$ kubectl apply -f examples/two-srl.c9s.yml
containerlab.topology.clabernetes/clab-srl02 created
```

### Verifying the deployment

Once the topology is deployed, clabernetes will do its magic. Without making quickstart too long,
let's just verify that it works:

Starting with listing `Containerlab` CRs in the `clabernetes` namespace we can see it is available:

```bash
$ kubectl get --namespace srl02 Containerlab
NAME         AGE
clab-srl02   26m
```

Looking in the Containerlab CR we can see that it took the topology definition from the `spec.config` field and split it to sub-topologies that are outlined in the `status.configs` section
of the resource:

```bash
kubectl get --namespace srl02 Containerlabs clab-srl02 -o yaml
```

```yaml
# --snip--
status:
  configs: |
    srl1:
        name: clabernetes-srl1
        prefix: null
        topology:
            nodes:
                srl1:
                    kind: srl
                    image: ghcr.io/nokia/srlinux:23.7.1
            links:
                - endpoints:
                    - srl1:e1-1
                    - host:srl1-e1-1
        debug: false
    srl2:
        name: clabernetes-srl2
        prefix: null
        topology:
            nodes:
                srl2:
                    kind: srl
                    image: ghcr.io/nokia/srlinux:23.7.1
            links:
                - endpoints:
                    - srl2:e1-1
                    - host:srl2-e1-1
```

The subtopologies are then deployed as deployments (which result in pods) in the cluster, and
containerlab running inside each pod deploys the topology:

```bash
$ kubectl get pods --namespace srl02 -o wide
NAME                               READY   STATUS    RESTARTS   AGE   IP           NODE           NOMINATED NODE   READINESS GATES
clab-srl02-srl1-77f7585fbc-m9v54   1/1     Running   0          31m   10.244.2.4   kind-worker    <none>           <none>
clab-srl02-srl2-54f8dddb88-hfftq   1/1     Running   0          31m   10.244.1.3   kind-worker2   <none>           <none>
```

We see that two pods are running, and more importantly, they run on different worker nodes.
These pods run containerlab inside in a docker-in-docker mode and each node deploys a subset of
the original topology. We can enter the pod and use containerlab CLI to verify the topology:

```bash
kubectl exec -n srl02 -it clab-srl02-srl1-77f7585fbc-m9v54  -- bash
```

And in the pod's shell we swim in the familiar containerlab waters:

```bash
root@clab-srl02-srl1-77f7585fbc-m9v54:/clabernetes# clab ins -a
+---+-----------+------------------+-----------------------------------+--------------+------------------------------+------+---------+----------------+----------------------+
| # | Topo Path |     Lab Name     |               Name                | Container ID |            Image             | Kind |  State  |  IPv4 Address  |     IPv6 Address     |
+---+-----------+------------------+-----------------------------------+--------------+------------------------------+------+---------+----------------+----------------------+
| 1 | topo.yaml | clabernetes-srl1 | clabernetes-clabernetes-srl1-srl1 | 0a16495fb358 | ghcr.io/nokia/srlinux:23.7.1 | srl  | running | 172.20.20.2/24 | 2001:172:20:20::2/64 |
+---+-----------+------------------+-----------------------------------+--------------+------------------------------+------+---------+----------------+----------------------+
```

### Accessing the nodes

## TODOs

- Make containerlab writing the deployment log to a file.
- Add ssh to the clab pods to enable local ssh access to the nodes.
- Watch docker process in container and restart if there is an issue
- Be very nice if we cached images maybe at the controller and/or somehow used cluster to pull
  image rather than having to pull every time container starts
- Generate kne binding for kne topos (and/or investigate doing that regardless of if topo is kne
  or not maybe... basically investigate just clab topo -> binding/testbed output so you can run
  fp tests against any topo)
