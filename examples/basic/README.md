# Basic Examples

This directory contains minimal examples to get started with Clabernetes.

## Examples

### simple-srl.yaml

The simplest possible Clabernetes topology - a single Nokia SR Linux node.

```bash
kubectl apply -f simple-srl.yaml
```

This creates:
- One launcher pod running the SR Linux container
- A LoadBalancer service exposing common management ports (SSH, gNMI, etc.)

### two-nodes-connected.yaml

Two SR Linux nodes connected via a point-to-point link.

```bash
kubectl apply -f two-nodes-connected.yaml
```

This creates:
- Two launcher pods, one for each SR Linux node
- VXLAN tunnels between the pods for the `e1-1` interface connection
- LoadBalancer services for each node

## Accessing Nodes

Once deployed, get the service IPs:

```bash
kubectl get svc -l clabernetes/topology=<topology-name>
```

SSH to a node (default credentials: admin/NokiaSrl1!):

```bash
ssh admin@<service-ip>
```

## Checking Status

View topology status:

```bash
kubectl get topologies
```

Check node readiness:

```bash
kubectl get topology <name> -o jsonpath='{.status.nodeReadiness}'
```

## Cleanup

```bash
kubectl delete -f <filename>.yaml
```
