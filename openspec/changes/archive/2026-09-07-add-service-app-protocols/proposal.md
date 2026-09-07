## Why

Expose Services currently identify only a port number and transport, leaving ingress controllers,
service meshes, and operators unable to distinguish HTTP, SSH, NETCONF, gNMI, or other application
protocols. Kubernetes `ServicePort.appProtocol` provides that hint, but c9s must assign standards-
compliant values without causing generic controllers to terminate device-owned TLS unexpectedly.

## What Changes

- Add standards-compliant `appProtocol` values to every port in the default auto-expose set,
  including IANA service names for standard protocols and `c9s.run/*` names for device gRPC
  protocols whose backend TLS identity must remain opaque to generic controllers.
- Let each Node override or suppress the effective application-protocol hint for a selected
  destination port, including selecting `kubernetes.io/h2c` for explicitly configured cleartext
  gRPC or `https` where TLS termination is acceptable.
- Accept an inheritable `c9s.run/appProtocols` Containerlab label directive and compile it into the
  generated Node's per-port overrides without copying the reserved directive into Kubernetes
  metadata.
- Reconcile `appProtocol` as controller-owned Service port state and document the default mapping,
  override syntax, TLS implications, and Kubernetes naming rules.
- Keep RDP and SFTP out of the default auto-expose set; port 22 remains `ssh`, including when the
  SSH File Transfer Protocol is used as an SSH subsystem.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `service-exposure`: Define default and overridden application-protocol hints on expose Service
  ports and require reconciliation of those hints.
- `node-lifecycle`: Let a Node declare application-protocol overrides associated with selected
  destination ports without changing port allocation or transport behavior.
- `topology-resource`: Compile the recognized `c9s.run/appProtocols` source directive into Node
  intent with normal Containerlab label inheritance and strict validation.

## Impact

- The additive `v1alpha1` Node API, generated CRDs, clients, and OpenAPI output gain per-port
  application-protocol override data.
- Default-port definitions, expose Service rendering/conformance, topology compilation, API and
  controller tests, generated Service fixtures, and exposure documentation are affected.
- Kubernetes automatically mirrors Service port `appProtocol` values into corresponding
  EndpointSlices; no new runtime dependency or device-plan behavior is introduced.
