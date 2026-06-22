# Scale effort — plain-language progress log

Short notes on what each phase changed and *why*. One glance = caught up.
Full design lives in `0001-scale-node-link-crds.md`.

**Big picture:** today a whole topology lives in a few giant objects, so K8s/etcd
(~1 MB per object) caps a lab at ~a few hundred nodes. The fix: split a `Topology`
into many small `Node` (and later `Link`) objects so nothing grows too big.

---

## Phase 0 — foundation (no behaviour change)

| What | Why |
|---|---|
| New `Node` and `Link` CRD types | The small per-node / per-link objects we split a topology into. |
| `ExpandTopology()` pure function | Turns one `Topology` → list of `Node`s + `Link`s. Pure = easy to test, changes nothing at runtime. |
| Unit tests + generated CRD/clientset | Prove the split is correct; wire the types into K8s. |

Result: types and logic exist but are **inert** — nothing runs yet.

---

## Phase 1 — make `Node` objects actually run (opt-in)

| What | Why |
|---|---|
| `decompose` boolean (on `Topology.spec.deployment`) | The **on/off switch**. `false` by default → existing labs behave exactly as before. Set `true` to try the new split path. Safe to ship. |
| `Node` status subresource | Lets the controller write a `Node`'s ready/not-ready into `.status` cleanly (standard K8s way to separate "what I want" from "what's happening"). |
| New `NodeController` (`controllers/node/`) | Watches `Node` objects and builds that node's ConfigMap + Deployment + Service(s) + PVC — i.e. one node's pod, on its own. |
| Per-node ConfigMap (was one big shared one) | This is the actual ceiling we remove: instead of one ConfigMap that grows with every node, each node gets its own small one. |
| `ReconcileNodes()` on the Topology | When `decompose=true`: expand the topology and create/update/delete the matching `Node` objects (the Topology becomes a manager, not a doer). |
| Still keeps the old `Connectivity` object | So tunnels between nodes keep working today. Splitting connectivity per-node is Phase 2. |

Result: with `decompose: true`, a topology runs as many independent `Node`s.
Default-off, so nobody is affected until they opt in.

**Not done yet (on purpose):** the `NodeController` still reads some shared
settings from the `Topology`, and two old big objects (`status.configs`, the
`Connectivity`) still exist. Those go away in Phase 2.

---

## Phase 2 — split connectivity per-node + Link ledger (opt-in)

| What | Why |
|---|---|
| Per-node `Connectivity` objects (was one per topology) | The last big object that grew with size. Now each node gets its own small `<topo>-<node>-connectivity` holding **only its own** tunnels — so nothing scales with topology size. |
| `Link` objects, one per cross-pod link | The durable, **distributed ledger** of tunnel-id allocations. Each `Link` carries its own `spec.tunnelID`, so the id list isn't a single growing object either. Built/pruned by the Topology (`ReconcileLinks`). |
| High-water-mark id allocation across per-node objects | Reads the ids already on the existing connectivity objects and **reuses** them (`AllocateTunnelIDs`). Renumbering a live tunnel would drop a working link, so ids are allocate-once. |
| `LAUNCHER_CONNECTIVITY_NAME` env + launcher reads it | The launcher now reads/watches **its own** connectivity object. Env unset = old behaviour (reads the topology-wide one), so the default path is untouched. The Node controller sets this env on the decomposed deployment. |
| Old monolithic `Connectivity` retired (decompose path) | When `decompose=true` the topology-wide object is pruned automatically (its ids are migrated into the per-node objects first). |

**Design choice:** per-node connectivity is written by the **Topology orchestrator**, not a
separate `controllers/link` reconciler. The orchestrator already computes the full tunnel data
(service-name destinations included) every reconcile, so a `LinkReconciler` would only re-derive it
and risk write contention on the shared per-node object. The `Link` objects are still created as the
id ledger / future status surface.

**Not done yet (on purpose):** `status.configs` (the other big legacy field) is still written — it
needs its consumers checked before removal — and the `Node` still reads a few shared knobs from the
`Topology`. Both are follow-ups (tracked below / Phase 4 polish).

Result: with `decompose: true`, a topology now runs as independent `Node`s **and** independent
per-node connectivity — no single object grows with topology size. Default-off.

---

## Phase 3 — indirect raw input (the last big object)

| What | Why |
|---|---|
| `spec.definition.containerlabRef` (ConfigMap **or** URL) | The raw clab YAML is the *last* whole-topology object — for thousands of nodes the string itself can blow the ~1MB ceiling. Now it can live in a ConfigMap / at a URL and the Topology only holds a tiny reference. |
| Controller resolves the ref and carries the body on `reconcileData.ResolvedDefinition` | The resolved (big) definition is held only in-memory for this reconcile; processors prefer it over the inline spec field. The original small-spec object is the one persisted — so the raw definition is **never written back** onto the Topology (which would re-create the ceiling). Carrying it on the reconcile data (not a deep-copied Topology) keeps a single Topology object, so status **conditions** written directly on it aren't lost. |
| No pipeline changes downstream | The processor reads `ResolvedDefinition` (or the inline field when empty), so every existing processor / `ExpandTopology` works unchanged. `Node` objects already carry their own per-node sub-config, so the Node controller never needs the raw input. |

Additive & **ungated** — works for inline and decomposed topologies alike. Mutually exclusive with
inline `containerlab`/`kne` (errors if both set).

**Deferred (documented):** `clabverter` does not yet auto-emit a ConfigMap ref for very large inputs
— it's an independent UX convenience (users can hand-write the ConfigMap + ref today) and would churn
the golden-file fixtures, so it's a separate follow-up.

---

## Phase 4 — status aggregation up to the Topology (decompose path)

| What | Why |
|---|---|
| `ReconcileNodeStatuses()` rolls owned `Node`s' readiness up to the Topology | In the decompose path the **Node controller** owns the deployments, so the Topology no longer sees them directly. It now reads each `Node.status.readiness` and writes it into the Topology's `status.nodeReadiness` — the same surface the monolithic path fills from its own deployments. |
| Shared `applyTopologyReadiness()` helper (used by both paths) | Computes the `TopologyReady` condition + `TopologyState` (Running/Degraded/Deploying) from the per-node map in one place, so decomposed and monolithic topologies report status identically. |
| Carry the resolved definition on `reconcileData` (not a deep-copied Topology) | Phase 3 originally processed a throwaway deep-copy; with status aggregation now writing **conditions** directly on the Topology, a copy would silently drop them when a `containerlabRef` is used. Refactored so there is a single Topology object end-to-end. |

**Result:** with `decompose: true`, the Topology's `.status` (node readiness, `TopologyReady`,
`TopologyState`) is populated from the owned `Node` objects — parity with the legacy path's status.

**Deliberately NOT done — needs real-cluster evidence, not code:**
- **Flipping the `decompose` default to `true`** — must wait until the decomposed path has soaked on a
  real cluster (the e2e / load-test boxes below are still unchecked). Flipping blind would change every
  user's behaviour with no soak data. **Recommendation: keep default `false`.**
- **e2e at scale / migration UX / docs** — require a live cluster to produce; tracked as open boxes.

---

# Checklist

Tick a box when it's implemented **and** verified (build + tests green).

### Phase 0 — inert foundation ✅

- [x] Design doc (`0001-scale-node-link-crds.md`)
- [x] `Node` / `Link` CRD types + scheme registration
- [x] Generated CRD YAML + deepcopy + clientset + openapi (`make run-generate`)
- [x] Pure `ExpandTopology` → `([]Node, []Link)` (`controllers/topology/expand.go`)
- [x] Expansion unit tests
- [x] **No runtime behaviour change** — verified inert

### Phase 1 — `NodeReconciler` + gated Topology fan-out 🔄

- [x] `decompose` gate on `Topology.spec.deployment` (default `false`) + CRD YAML
- [x] `Node` status subresource + CRD YAML
- [x] `controllers/node` package (`Controller` / `Reconcile` / `Reconciler`)
- [x] Per-node ConfigMap + Deployment + fabric Service + expose Service + PVC, reusing the existing
  Topology sub-reconcilers
- [x] `Node.status.ready` from the Deployment's `Available` condition
- [x] `ReconcileNodes` on the Topology — expand → create/update/prune owned `Node`s (gated); still
  reconciles the old `Connectivity` so tunnels form
- [x] `NodeController` registered in `manager/start.go`
- [x] RBAC — covered by the existing manager `*` rule on `clabernetes.containerlab.dev`
- [x] `go build ./...` + topology tests green
- [ ] `envtest`/unit coverage for the `NodeReconciler`
- [ ] e2e: a decomposed topology boots and forms tunnels on a real cluster
- [ ] Load-test the reconcile fan-out

### Phase 2 — per-node connectivity + `Link` ledger 🔄

- [x] Per-node `Connectivity` objects written by the Topology (`ReconcilePerNodeConnectivity`)
  — replaces a separate `controllers/link` reconciler on purpose (see design choice above)
- [x] High-water-mark allocation across the per-node objects; create/prune `Link` objects
  (`ReconcileLinks`)
- [x] Migrate launcher watch + startup read to its own node's connectivity object
  (`LAUNCHER_CONNECTIVITY_NAME`, fallback to topology-wide)
- [x] Retire the monolithic `Connectivity` in the decompose path (pruned, ids migrated first)
- [x] `go build ./...` + `go vet` + topology tests green
- [ ] Drop `status.configs` (deferred — check consumers first)
- [ ] Richer self-contained `NodeSpec` so the `NodeReconciler` stops fetching the `Topology`
- [ ] `envtest`/e2e: decomposed topology boots and forms tunnels via per-node connectivity

### Phase 3 — indirect raw input 🔄

- [x] `spec.definition.containerlabRef` (ConfigMap **or** URL) for the raw input
  (`controllers/topology/definitionref.go`) — resolved body carried on `reconcileData.ResolvedDefinition`,
  never persisted back; API type + deepcopy + topology CRD YAML
- [x] `go build ./...` + `go vet` + topology/apis/clabverter tests green
- [ ] `clabverter` emits a reference instead of inline for very large inputs (deferred — UX
  convenience, golden-fixture churn)

### Phase 4 — status aggregation up to the Topology 🔄

- [x] `ReconcileNodeStatuses` rolls owned `Node` readiness into the Topology status (decompose path)
- [x] Shared `applyTopologyReadiness` (`TopologyReady` condition + `TopologyState`) used by both paths
- [x] Single-Topology refactor (resolved definition on `reconcileData`, no deep-copy) so status
  conditions aren't lost with `containerlabRef`
- [x] `go build ./...` + `go vet` + topology tests green
- [ ] e2e at scale: decomposed topology status matches the legacy path on a real cluster
- [ ] Migration UX + user docs
- [ ] **Flip the `decompose` default** — blocked on the soak/e2e/load-test boxes above (recommend
  keeping default `false` until then)
