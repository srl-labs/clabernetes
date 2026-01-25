# Nokia SR-SIM Support Guide

This guide explains how to deploy Nokia SR-SIM (SR OS Simulator) topologies with Clabernetes, including supported configurations.

## Overview

Nokia SR-SIM is a containerized version of Nokia SR OS, replacing the legacy VM-based vSIM variant (`vr-sros`). SR-SIM is identified by the `nokia_srsim` kind in containerlab topology files.

## Prerequisites

1. **License**: A valid SR-SIM license file is mandatory. The license must be provided via the `license` directive or the deployment will fail.

2. **Image**: The SR-SIM container image must be downloaded from the Nokia Support Portal and loaded into your container runtime.

3. **Resources**: SR-SIM nodes require significant resources. Ensure your cluster nodes have adequate CPU and memory.

## Supported Configurations

### Integrated Systems

Integrated SR-SIM systems run as a single container:

| Platform Type | Description |
|---------------|-------------|
| `sr-1` | SR-1 integrated system (default) |
| `sr-1s` | SR-1s integrated system |

**Example topology:**

```yaml
apiVersion: clabernetes.containerlab.dev/v1alpha1
kind: Topology
metadata:
  name: srsim-integrated
spec:
  definition:
    containerlab: |
      name: srsim-integrated
      topology:
        kinds:
          nokia_srsim:
            image: nokia_srsim:25.7.R1
            license: /path/to/license.txt
        nodes:
          sr1:
            kind: nokia_srsim
            type: sr-1
            startup-config: sr1-config.cfg
          sr2:
            kind: nokia_srsim
            type: sr-1s
```

### Distributed Chassis Systems

Distributed chassis-based SR-SIM systems (SR-7, SR-14s, etc.) use multiple containers that share a network namespace via the `network-mode: container:<name>` directive.

| Platform Type | Description |
|---------------|-------------|
| `sr-2s` | SR-2s chassis system |
| `sr-2se` | SR-2se chassis system |
| `sr-7` | SR-7 chassis system |
| `sr-14s` | SR-14s chassis system |
| `sr-1x-92S` | SR-1x-92S system |

**How it works:**

Clabernetes automatically detects nodes with `network-mode: container:<primary-node>` and groups them together. All nodes in a group are deployed in the same launcher pod, allowing them to share the network namespace as required by distributed SR-SIM.

**Example distributed topology:**

```yaml
apiVersion: clabernetes.containerlab.dev/v1alpha1
kind: Topology
metadata:
  name: srsim-distributed
spec:
  deployment:
    resources:
      # Resources are specified for the primary node (group leader)
      srsim-a:
        requests:
          memory: "8Gi"
          cpu: "4"
        limits:
          memory: "16Gi"
          cpu: "8"
  definition:
    containerlab: |
      name: srsim-distributed
      topology:
        kinds:
          nokia_srsim:
            image: nokia_srsim:25.7.R1
            license: /opt/nokia/sros/license.txt
        nodes:
          # Primary node (CPM-A) - this is the "group leader"
          srsim-a:
            kind: nokia_srsim
            type: sr-7
            env:
              NOKIA_SROS_SLOT: A
          # Secondary node (CPM-B) - references primary via network-mode
          srsim-b:
            kind: nokia_srsim
            type: sr-7
            network-mode: container:srsim-a
            env:
              NOKIA_SROS_SLOT: B
          # IOM slot - also references primary
          srsim-iom1:
            kind: nokia_srsim
            type: sr-7
            network-mode: container:srsim-a
            env:
              NOKIA_SROS_SLOT: "1"
        links:
          # Links to external nodes work normally
          - endpoints: ["srsim-iom1:1/1/c1/1", "external-router:e1-1"]
```

**Key points for distributed mode:**

1. **Primary node**: The node without `network-mode` is the primary (group leader). Resources, services, and tunnels are associated with this node.

2. **Secondary nodes**: Nodes with `network-mode: container:<primary>` are grouped with their primary and deployed in the same pod.

3. **Links**: Links between nodes in the same group remain as local containerlab links. Links to nodes outside the group use VXLAN tunnels.

4. **Service names**: Services are created for the primary node only. Use the primary node name when connecting from other pods.

### MDA and Component Configuration

For integrated systems, you can customize MDAs (Media Dependent Adapters) using environment variables:

```yaml
nodes:
  sr1:
    kind: nokia_srsim
    type: sr-1
    env:
      NOKIA_SROS_MDA_1: me6-100gb-qsfp28
      NOKIA_SROS_MDA_2: me12-10/1gb-sfp+
```

Or using the `components` block:

```yaml
nodes:
  sr1:
    kind: nokia_srsim
    type: sr-1
    components:
      - slot: 1
        env:
          NOKIA_SROS_MDA_1: me6-100gb-qsfp28
```

## Interface Naming

SR-SIM uses a specific interface naming convention:

```
L/xX/M/cC/P
```

- `L` - Line card slot
- `X` - MDA slot (optional for some platforms)
- `M` - MDA number
- `C` - Connector number
- `P` - Port number

**Example:** `1/1/c1/1` = Card 1, MDA 1, Connector 1, Port 1

In topology links:

```yaml
links:
  - endpoints: ["sr1:1/1/c1/1", "sr2:1/1/c1/1"]
```

## Resource Recommendations

SR-SIM nodes are resource-intensive. Configure appropriate resource limits:

```yaml
apiVersion: clabernetes.containerlab.dev/v1alpha1
kind: Topology
metadata:
  name: srsim-with-resources
spec:
  deployment:
    resources:
      sr1:
        requests:
          memory: "4Gi"
          cpu: "2"
        limits:
          memory: "8Gi"
          cpu: "4"
  definition:
    containerlab: |
      name: srsim
      topology:
        nodes:
          sr1:
            kind: nokia_srsim
            type: sr-1
```

For distributed systems, specify resources for the primary node:

```yaml
spec:
  deployment:
    resources:
      srsim-a:  # Primary node name
        requests:
          memory: "8Gi"
          cpu: "4"
        limits:
          memory: "16Gi"
          cpu: "8"
```

## License File Mounting

The license file must be accessible to the SR-SIM container. Use ConfigMaps to mount the license:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: srsim-license
data:
  license.txt: |
    # Your license content here

---
apiVersion: clabernetes.containerlab.dev/v1alpha1
kind: Topology
metadata:
  name: srsim-with-license
spec:
  deployment:
    filesFromConfigMap:
      sr1:
        - filePath: /opt/nokia/sros/license.txt
          configMapName: srsim-license
          configMapPath: license.txt
  definition:
    containerlab: |
      name: srsim
      topology:
        kinds:
          nokia_srsim:
            license: /opt/nokia/sros/license.txt
        nodes:
          sr1:
            kind: nokia_srsim
            type: sr-1
```

## Related Resources

- [Containerlab SR-SIM Documentation](https://containerlab.dev/manual/kinds/sros/)
- [SR-SIM Lab Examples](https://github.com/srl-labs/containerlab/tree/main/lab-examples/sr-sim)
- [File Mounting Guide](./file-mounting.md)
- [Resource Management Guide](./resource-management.md)
