## ADDED Requirements

### Requirement: Node declares per-port application-protocol intent

A Node SHALL accept an optional `spec.appProtocols` keyed list whose entries contain `port` and
`appProtocol`. Each `port` MUST use the canonical `<port>/<transport>` form, where the port number
is between 1 and 65535 and transport is lowercase `tcp` or `udp`; duplicate ports MUST be rejected.
Each non-empty `appProtocol` MUST follow Kubernetes qualified-name syntax, while an explicitly empty
value SHALL mean that no application-protocol hint is desired. The list SHALL describe Service
metadata only and MUST NOT alter the device plan, application configuration, transport protocol, or
set of exposed ports.

#### Scenario: Override one selected port

- **WHEN** a Node declares an entry with `port: 57400/tcp` and
  `appProtocol: kubernetes.io/h2c`, and port 57400/TCP is selected for exposure
- **THEN** the Node's expose Service uses that application-protocol value for the matching port

#### Scenario: Reject a malformed destination key

- **WHEN** a Node application-protocol entry uses a host binding, port range, out-of-range port,
  uppercase transport, or unsupported transport in its `port` field
- **THEN** the Node manifest is rejected as invalid

#### Scenario: Reject an invalid application-protocol name

- **WHEN** a Node application-protocol entry contains a non-empty value that violates Kubernetes
  qualified-name syntax
- **THEN** the Node manifest is rejected as invalid

#### Scenario: Reject a duplicate destination key

- **WHEN** a Node declares more than one application-protocol entry for the same canonical port
- **THEN** the Node manifest is rejected as invalid

#### Scenario: Keep application protocol out of device planning

- **WHEN** two otherwise identical Nodes differ only in `spec.appProtocols`
- **THEN** they produce equivalent device application plans and differ only in expose Service
  metadata
