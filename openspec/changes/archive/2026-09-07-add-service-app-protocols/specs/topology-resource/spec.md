## MODIFIED Requirements

### Requirement: Containerlab node labels become Kubernetes labels

The compiler SHALL carry Containerlab Node labels onto the emitted Node's object metadata,
inheriting them from topology defaults and kinds the same way Node environment variables are
inherited. The Node controller SHALL propagate a Node's labels to its direct workload and Pods,
excluding labels in the reserved `c9s.run/` namespace and controller-owned label keys, without
altering the workload's Pod selector.

A label Kubernetes would reject, a reserved label, or a controller-owned key MUST be rejected under
the direct runtime's no-semantic-loss compilation contract unless it is a recognized source
directive. The `c9s.run/exposePorts` directive SHALL be consumed into `spec.ports`, and the
`c9s.run/appProtocols` directive SHALL be consumed into `spec.appProtocols`. Neither directive may
be copied to object metadata.

#### Scenario: Label a lab node

- **WHEN** a source topology Node declares a valid Containerlab label
- **THEN** the emitted Node and direct workload carry it so the Pod can be selected by it

#### Scenario: Labels inherit from defaults and kinds

- **WHEN** labels are declared at topology defaults, kind, and Node level
- **THEN** the emitted Node carries the merged set with the most specific value winning

#### Scenario: Omit a label Kubernetes cannot accept

- **WHEN** a source topology Node declares a label whose key or value is invalid as a Kubernetes
  label
- **THEN** compilation fails with a diagnostic naming it instead of silently dropping metadata

#### Scenario: Omit a label in the reserved namespace

- **WHEN** a source topology Node declares a label in the `c9s.run/` namespace other than a
  recognized source directive
- **THEN** compilation fails because user input cannot set controller behavior implicitly

#### Scenario: Declare c9s-only service ports without publishing Docker host ports

- **WHEN** a source topology Node declares `c9s.run/exposePorts: "9273/tcp,8125/udp"`
- **THEN** the emitted Node carries both entries in `spec.ports`, the directive is absent from
  metadata, and equivalent ordinary entries are not duplicated

#### Scenario: Reject an invalid c9s expose ports directive

- **WHEN** the expose-ports directive contains an empty or malformed entry
- **THEN** compilation fails with a diagnostic naming the Node, label, and invalid entry

#### Scenario: Inherit c9s-only service ports

- **WHEN** `c9s.run/exposePorts` is declared on topology defaults or a kind
- **THEN** every effective Node receives its canonical ports subject to normal label override
  semantics and no emitted Node carries the directive in metadata

#### Scenario: Declare application protocols from a topology

- **WHEN** a source topology Node declares
  `c9s.run/appProtocols: "57400/tcp=kubernetes.io/h2c,443/tcp=https"`
- **THEN** the emitted Node carries both keyed entries in `spec.appProtocols` and the directive is
  absent from metadata

#### Scenario: Suppress a default application protocol from a topology

- **WHEN** a source topology Node declares `c9s.run/appProtocols: "57400/tcp="`
- **THEN** the emitted Node carries an entry for `57400/tcp` with an explicitly empty
  `appProtocol`

#### Scenario: Reject an invalid application-protocol directive

- **WHEN** an application-protocol directive has an empty entry, lacks the `=` separator, contains
  a malformed destination key, repeats a canonical destination key, or contains an invalid
  non-empty application-protocol value
- **THEN** compilation fails with a diagnostic naming the Node, label, and invalid entry

#### Scenario: Inherit application-protocol overrides

- **WHEN** `c9s.run/appProtocols` is declared on topology defaults, a kind, or a group
- **THEN** each effective Node receives the entries selected by normal label override semantics and
  no emitted Node carries the directive in metadata

#### Scenario: Application-protocol directive does not expose a port

- **WHEN** a valid application-protocol directive names a port that is neither automatically nor
  explicitly selected
- **THEN** the emitted entry remains descriptive intent and does not add the port to `spec.ports`

#### Scenario: Preserve exposure policy

- **WHEN** a valid source directive is compiled for a Topology whose effective NodeProfile disables
  exposure or auto-exposure
- **THEN** the directive contributes only its defined Node port or application-protocol intent and
  the profile continues to control whether a Service and automatic ports are realized

#### Scenario: Omit a controller-owned label key

- **WHEN** a source topology Node declares a key such as `app.kubernetes.io/name` that c9s owns for
  identity or selection
- **THEN** compilation fails rather than allowing it to overwrite a controller invariant
