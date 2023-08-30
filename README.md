[![Discord](https://img.shields.io/discord/860500297297821756?style=flat-square&label=discord&logo=discord&color=00c9ff&labelColor=bec8d2)](https://discord.gg/vAyddtaEV9)
[![Go Report](https://img.shields.io/badge/go%20report-A%2B-blue?style=flat-square&color=00c9ff&labelColor=bec8d2)](https://goreportcard.com/report/github.com/srl-labs/clabernetes)

clabernetes
===========

Love containerlab? Want containerlab, just distributed in a kubernetes cluster? Enter 
clabernetes -- containerlab + kubernetes. clabernetes is a kubernetes controller that deploys valid 
containerlab topologies into a kubernetes cluster.


## Requirements

- A kubernetes cluster of course!
  - Requires `MixedProtocolLBService` feature gate (stable in 1.26, enabled in 1.24)
- The ability to run privileged pods in your cluster
- A load balancer in your cluster if you want to expose nodes "directly"
- Sufficient privileges to install the controller


## Installation

```
helm upgrade --install clabernetes oci://ghcr.io/srl-labs/clabernetes/clabernetes
```


## A Simple Example

```yaml
---
apiVersion: topology.clabernetes/v1alpha1
kind: Containerlab
metadata:
  name: clab-srl02
  namespace: clabernetes
spec:
  config: |-
    name: srl02
    topology:
      nodes:
        srl1:
          kind: srl
          image: ghcr.io/nokia/srlinux
        srl2:
          kind: srl
          image: ghcr.io/nokia/srlinux
      links:
        - endpoints: ["srl1:e1-1", "srl2:e1-1"]
```


## TODOs

- Watch docker process in container and restart if there is an issue
- Be very nice if we cached images maybe at the controller and/or somehow used cluster to pull 
  image rather than having to pull every time container starts
- Generate kne binding for kne topos (and/or investigate doing that regardless of if topo is kne 
  or not maybe... basically investigate just clab topo -> binding/testbed output so you can run 
  fp tests against any topo)