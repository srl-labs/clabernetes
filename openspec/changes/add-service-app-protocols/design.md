## Context

See `proposal.md` for motivation and the delta specs for the behavioral contract. The default
management ports are currently assembled as `NodeExposedPort` values containing only destination,
publication port, and transport. Direct reconciliation first merges planned image ports, explicit
Node ports, and automatic defaults, then renders an expose Service from that merged allocation.
The merge intentionally discards each port's source, and Service conformance currently compares
name, port, transport, and target but not `appProtocol`.

`Node.spec` embeds the supported Containerlab node vocabulary and separately carries c9s-specific
per-node intent. Deployment policy, including whether and how a Service is created, belongs to a
NodeProfile. A port's application identity is instead intrinsic to the Node endpoint and can vary
between Nodes using the same profile. The topology compiler already handles c9s-only per-node
intent through reserved Containerlab label directives that are consumed before Kubernetes metadata
validation.

Kubernetes represents `ServicePort.AppProtocol` as an optional string pointer and mirrors it to
EndpointSlices. The value is a hint only: c9s does not create an application proxy or TLS
termination point.

## Goals / Non-Goals

**Goals:**

- Resolve one deterministic application-protocol hint after all port sources have been merged.
- Keep default protocol definitions and their application hints in one reviewable table.
- Give direct Node and source Topology authors the same per-node override and suppression behavior.
- Reject malformed override structure before it can produce an invalid Service.
- Converge existing owned Services when desired application-protocol metadata changes.

**Non-Goals:**

- Infer application protocols from Node kind, image name, credentials, startup configuration, or
  observed traffic.
- Configure device TLS, terminate TLS, or teach an ingress implementation how to interpret
  `c9s.run/*` values.
- Change the auto-exposed port set or add RDP, SSH File Transfer Protocol, or another endpoint.
- Add application-protocol metadata to internal fabric, alias, or management-mesh Services.
- Add a Service-wide semantic label; one expose Service contains unrelated application ports, so a
  single label cannot describe them accurately.

## Decisions

### 1. Resolve application protocol from the final port tuple

Represent each built-in management endpoint internally as destination port, transport, and default
application protocol. Derive the existing allocation and management-translation inputs from this
table, and resolve the hint by canonical `<destination>/<transport>` key while rendering each final
Service port. Apply a Node override after the built-in lookup.

This deliberately makes a matching explicit or image-declared port receive the same default hint
as an automatically added port. The current merge lets an explicit or image source claim a tuple
before defaults are added; carrying metadata only on automatically created allocation objects would
therefore make the result depend on source ordering. A user reusing a conventional number for a
different application can replace or suppress the hint explicitly.

Alternative considered: add `AppProtocol` to device-plan ports or Node allocation status and carry
it through every merge. That expands runtime and observed-status contracts for metadata that only
the Kubernetes Service renderer consumes, while still requiring conflict precedence rules at every
source boundary.

### 2. Put overrides on Node intent as a schema-valid keyed list

Add `spec.appProtocols` outside the embedded `NodeDefinition`. It is a Kubernetes keyed list of
entries with:

- `port`: required canonical `<number>/tcp` or `<number>/udp`, used as the list-map key.
- `appProtocol`: required string; a valid non-empty Kubernetes qualified name replaces the default,
  while an empty string suppresses the field.

List-map semantics make duplicate canonical ports invalid at the API boundary. Field validation can
reuse the existing bounded port-range expression and a qualified-name-compatible expression, with
empty application protocol retained as the explicit suppression value. Keeping the field outside
`NodeDefinition` prevents it from becoming unsupported Containerlab vocabulary or planner input.

Alternative considered: a `map[string]string`. It is compact but Kubernetes structural schemas do
not provide the same straightforward per-key validation and uniqueness semantics as a keyed list.
Alternative considered: place the map on NodeProfile. That conflates endpoint identity with
realization policy and cannot naturally represent two Nodes sharing one exposure profile but using
different gRPC TLS modes.

### 3. Compile a consumed Containerlab label directive

Recognize `c9s.run/appProtocols` as a source-only directive. Its value is a comma-separated list of
`<port-definition>=<appProtocol>` entries, for example:

```yaml
labels:
  c9s.run/appProtocols: "57400/tcp=kubernetes.io/h2c,443/tcp=https"
```

The source-side port definition accepts the same number and optional TCP/UDP spelling as existing
port directives, then normalizes it to the canonical lowercase key required by the Node API. An
empty right-hand side is retained as suppression. Missing separators, empty entries, malformed
ports, invalid non-empty values, and duplicates after normalization produce compiler diagnostics.

Consume the directive after Containerlab defaults, kinds, groups, and Node labels have been
flattened. Store parsed entries in a Node-keyed field on the compiler's intermediate result, remove
the reserved label, and copy the entries into each emitted Node. The directive follows ordinary
Containerlab label override semantics: a more specific label replaces the less specific directive
value as a whole.

Alternative considered: add an application-protocol field inside the embedded Containerlab YAML.
That would cease to be valid Containerlab vocabulary and violate the compiler's compatibility
boundary. Alternative considered: add a separate node-keyed Topology API map. The consumed label is
already the established mechanism for c9s-only endpoint declarations, keeps intent beside the
source Node, and inherits through Containerlab scopes without another parallel inheritance model.

### 4. Keep user-value validation bounded and Kubernetes-compatible

The generated Node schema rejects malformed canonical port keys, duplicate entries, and values
that cannot be Kubernetes qualified names. The topology compiler applies equivalent checks before
emitting resources. c9s owns and reviews its built-in unprefixed values against the IANA registry;
the documentation states that user-supplied unprefixed values are reserved for IANA service names
and that all other custom values require a DNS prefix.

c9s will not embed or dynamically retrieve the complete IANA registry merely to validate arbitrary
user overrides. Kubernetes itself validates qualified-name syntax but does not provide a local API
for proving IANA registration, so registry-semantic correctness beyond c9s-owned defaults remains
part of the declarative input contract.

Alternative considered: allow only a small c9s-maintained enum. That would reject legitimate IANA
or implementation-prefixed protocols and make the override less useful without improving the
validity of values outside the enum.

### 5. Reconcile `AppProtocol` with the rest of each Service port

Set `ServicePort.AppProtocol` only when the resolved value is non-empty. Extend Service conformance
to compare the pointer value for the matching named port. Existing Service update behavior then
adds, changes, or removes hints while retaining API-server-assigned fields such as NodePort. No
Service recreation is required because `appProtocol` is mutable.

Alternative considered: leave conformance unchanged and rely on another Service difference to
trigger an update. Existing Services would remain indefinitely without hints, and override changes
could be ignored, so this does not meet convergence requirements.

## Risks / Trade-offs

- [A conventional port number carries a different application] -> The per-port override can replace
  the built-in hint or explicitly suppress it.
- [A generic controller treats `https` as permission to terminate TLS] -> Device gRPC defaults use
  `c9s.run/*`; `https` appears only through explicit user intent.
- [A custom unprefixed override is syntactically valid but not IANA-registered] -> Keep all c9s
  defaults registry-compliant, document the Kubernetes reservation, and require prefixes for custom
  examples rather than vendoring a stale registry snapshot.
- [The source directive mini-language produces ambiguous duplicate entries] -> Normalize first and
  reject duplicate canonical port keys instead of applying last-value-wins behavior.
- [Generated APIs drift from source types] -> Regenerate and verify CRDs, clients, deepcopy code,
  and OpenAPI output with the repository generation target.
- [Service updates lose allocated NodePorts] -> Exercise app-protocol drift through the existing
  reconciliation path and assert NodePort preservation.

## Migration Plan

1. Regenerate and install the additive Node CRD schema before or with the controller release.
2. Roll out the controller; existing Nodes with no overrides resolve built-in defaults, and their
   owned expose Services converge during normal reconciliation.
3. Apply optional Topology directives or direct Node entries where device configuration requires
   cleartext gRPC, terminable TLS, another qualified protocol, or suppression.

Rolling back only the controller leaves the harmless Service and Node metadata present, although
the old controller no longer reconciles `appProtocol`. Rolling back the CRD schema as well requires
removing `spec.appProtocols` from declarative Node inputs; the Service hints may remain until another
old-controller update rewrites those ports.
