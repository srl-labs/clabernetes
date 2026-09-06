## 1. Node API Contract

- [ ] 1.1 Add the c9s-specific `spec.appProtocols` keyed-list source types outside
  `NodeDefinition`, including canonical port, uniqueness, qualified-name-or-empty validation, and
  explanatory API comments; verify focused API schema tests accept valid overrides and suppression
  while rejecting malformed ports, duplicate keys, and invalid values.
- [ ] 1.2 Regenerate deepcopy code, clients, OpenAPI output, and CRD copies from the source API and
  verify `make verify-generated` reports no drift.

## 2. Topology Compilation

- [ ] 2.1 Add the reserved `c9s.run/appProtocols` directive and parser, normalize its port keys with
  existing destination-port rules, preserve empty suppression values, and return deterministic
  diagnostics for malformed or duplicate entries; verify focused compiler tests cover valid,
  invalid, uppercase/default-TCP normalization, and suppression inputs.
- [ ] 2.2 Carry parsed application-protocol entries through the compiled-topology intermediate into
  generated Nodes while consuming the source label; verify compiler and renderer tests cover
  defaults/kinds/groups/Node inheritance, renamed Nodes, absence from emitted metadata, and the rule
  that the directive does not add `spec.ports` entries.

## 3. Service Resolution

- [ ] 3.1 Extend the built-in management-port definition with the exact application-protocol table
  from the service-exposure spec while deriving the existing allocation and management-translation
  inputs from the same source; verify controller tests assert every mapping and prove the exposed
  port set and transport values are unchanged.
- [ ] 3.2 Resolve each final Service port's hint by destination and transport, then apply Node
  replacement or suppression intent without changing device plans or port selection; verify focused
  rendering tests cover defaults, explicit and OCI claims of default ports, unknown ports,
  `kubernetes.io/h2c`, `https`, custom prefixed values, and empty suppression.
- [ ] 3.3 Include `AppProtocol` in expose Service conformance and ordinary updates; verify
  reconciliation tests add, change, and remove the hint on existing Services while preserving
  Kubernetes-assigned NodePorts.

## 4. Documentation And Fixtures

- [ ] 4.1 Update the exposure guide and topology concept documentation with the default mapping,
  direct Node keyed-list syntax, `c9s.run/appProtocols` syntax and inheritance, IANA/Kubernetes
  naming rules, TLS pass-through guidance, and the SSH-versus-SFTP distinction; verify
  `make check-docs` succeeds.
- [ ] 4.2 Update generated-Service golden fixtures to include the expected hints only on known
  ports and verify the relevant fixture assertions or focused e2e package tests pass without
  requiring a live cluster.

## 5. Repository Validation

- [ ] 5.1 Run `go test ./apis/v1alpha1/... ./compiler/... ./controllers/node/...` and fix any focused
  regressions in API, compiler, and Service behavior.
- [ ] 5.2 Run `make test`, `make lint`, `make check-docs`, and `make verify-generated`; inspect the
  resulting diff and verify only intended source, generated, test, fixture, documentation, and
  OpenSpec files changed.
