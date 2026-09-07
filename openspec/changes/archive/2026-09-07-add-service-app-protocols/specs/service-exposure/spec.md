## ADDED Requirements

### Requirement: Expose Service ports carry application-protocol hints

For every selected Service port matching a member of the default management-port set by
destination port and transport, the system SHALL set `Service.spec.ports[].appProtocol` to the
following value. The mapping SHALL apply regardless of whether the selected port originated from
automatic exposure, an explicit Node declaration, or imported image metadata.

| Destination port | Transport | `appProtocol` |
| ---: | --- | --- |
| 21 | TCP | `ftp` |
| 22 | TCP | `ssh` |
| 23 | TCP | `telnet` |
| 80 | TCP | `http` |
| 161 | UDP | `snmp` |
| 443 | TCP | `https` |
| 830 | TCP | `netconf-ssh` |
| 5000 | TCP | `telnet` |
| 5900 | TCP | `rfb` |
| 6030 | TCP | `c9s.run/gnmi` |
| 9339 | TCP | `c9s.run/gnmi` |
| 9340 | TCP | `c9s.run/gribi` |
| 9559 | TCP | `c9s.run/p4runtime` |
| 57400 | TCP | `c9s.run/gnmi` |

The system MUST use unprefixed names only for the corresponding IANA service, MUST use the
Kubernetes-defined `kubernetes.io/h2c` value only when explicitly selected for cleartext HTTP/2,
and MUST retain an implementation-defined prefix for the default device gRPC application hints.
Application-protocol hints MUST NOT change port selection, transport, routing, TLS termination, or
device traffic.

#### Scenario: Render the standard protocol defaults

- **WHEN** an expose Service contains ports from the default management-port set
- **THEN** each matching Service port carries the exact `appProtocol` value in the default mapping

#### Scenario: Preserve a hint when an explicit source claims a default port

- **WHEN** a Node declaration or imported image port claims the same destination and transport as
  a default management port
- **THEN** the resulting single Service port retains the mapped default `appProtocol`

#### Scenario: Identify the QEMU console by its actual protocol

- **WHEN** the expose Service contains the vrnetlab QEMU console on port 5000/TCP
- **THEN** its `appProtocol` is `telnet` rather than the IANA service assigned to port 5000

#### Scenario: Identify SSH File Transfer traffic as SSH

- **WHEN** the expose Service contains port 22/TCP
- **THEN** its `appProtocol` is `ssh` and no separate SFTP application protocol is asserted

#### Scenario: Leave an unknown port unspecified

- **WHEN** a selected Service port has no built-in mapping and no Node override
- **THEN** the Service port omits `appProtocol`

### Requirement: Node application-protocol intent overrides defaults

The system SHALL resolve a Node's application-protocol entry for a selected destination port and
transport after merging all port sources. A non-empty entry SHALL replace the built-in value, while
an explicitly empty entry SHALL suppress `appProtocol` for that Service port. An entry for a port
that is not selected SHALL have no effect and MUST NOT cause that port to be exposed.

#### Scenario: Mark cleartext gRPC

- **WHEN** a selected gNMI port has a Node override of `kubernetes.io/h2c`
- **THEN** the rendered Service port uses `kubernetes.io/h2c` instead of `c9s.run/gnmi`

#### Scenario: Mark terminable TLS

- **WHEN** a selected gRPC port has a Node override of `https`
- **THEN** the rendered Service port uses `https`

#### Scenario: Suppress a built-in hint

- **WHEN** a selected default port has an explicitly empty Node override
- **THEN** the rendered Service port omits `appProtocol`

#### Scenario: Override does not select a port

- **WHEN** a Node has an application-protocol entry for a destination and transport that is not
  otherwise selected for exposure
- **THEN** no Service port is added for that entry

### Requirement: Application-protocol drift is reconciled

The system SHALL treat each rendered Service port's `appProtocol` as controller-owned desired
state. It MUST update an owned expose Service when the current hint differs from the resolved hint,
including adding, changing, or removing the field, without discarding Kubernetes-assigned Service
fields that are preserved during ordinary Service updates.

#### Scenario: Add hints to an existing Service

- **WHEN** an owned expose Service created before this capability lacks the desired application-
  protocol hints
- **THEN** reconciliation updates its matching Service ports with those hints

#### Scenario: Apply a changed override

- **WHEN** a Node changes a port override from `c9s.run/gnmi` to `kubernetes.io/h2c`
- **THEN** reconciliation changes the matching Service port's `appProtocol` and preserves its
  Kubernetes-assigned NodePort when applicable
