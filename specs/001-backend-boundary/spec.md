# Feature Specification: Backend Boundary

**Feature Branch**: `001-backend-boundary`

**Created**: 2026-09-03

**Status**: Draft

**Input**: User description: "Backend boundary: game-mode skeleton plus the localhost backend contract — version check at boot, health, typed command envelope with operation id and revision, accepted/refused/already-applied answers, fail-closed refusal, audit row. The first consequential record carried across it is the wallet."

## Current State

No game code exists. The addon folder holds only the project file. `RULES.md` v0.20 and
`TECHNICAL-DESIGN.md` v1.0 are written; the constitution (2026-09-03) moves the system
of record for logical records from the native save bundle to a service on the same
machine. This feature is the first one that touches that boundary, so it carried the
matching technical-design revision (see Source Revisions).

Two engine facts are established by the shipped lobby addon on this community's
servers: the game runtime can make outbound HTTP calls from the server, and a player's
stable identity is only reliable after the connection's audit-success event. Nothing
in the official scripts provides an inbound listener, so the game must always be the
caller.

## Clarifications

### Session 2026-09-03

- Q: How do players and operators issue this feature's commands inside the game? → A:
  a minimal in-game cockpit panel (boundary status, wallet lookup, credit/debit with
  reason, recent audit entries) for operators, and a balance display for players.
- Q: What counts as a known contract version? → A: exact match; each game build knows
  exactly one contract version and refuses every other.
- Q: May one player have more than one consequential command in flight? → A: no; a
  second command while one is pending is refused at once with a stable "busy" reason
  and no backend call.

### Session 2026-09-04

- Q: The local dedicated server has no backend reach, so every player audits with an
  empty identity and the whole feature is refused; real servers always give identities.
  What now? → A: a development flag in the game's backend config
  (`DevIdentityFromName`, default off) substitutes an identity derived from the player
  name, using the same derivation the engine applies to non-dedicated play. Off, the
  fail-closed behaviour is unchanged.
- Q: A receipt pushed live to an online player was replayed on every later connect.
  How is delivery settled? → A: the compensation carries whether the target has a
  live session; when it does, the backend records the receipt as delivered at
  creation, so connect replays only receipts created while the player was away. No
  extra call. The website reads the ledger and is unaffected. Accepted
  consequence: a receipt whose command was in flight when the game server died is
  never shown; the ledger row stands.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - The server proves its backend before letting money move (Priority: P1)

The operator starts the dedicated server. Before any consequential command is accepted,
the game confirms that the backend is reachable on the same machine and speaks a
contract version the game knows. The operator can see, from inside the game, whether
the boundary is ready and, if not, the exact reason.

**Why this priority**: every later feature (wallet, faction, vehicles, plots, auction)
depends on this answer. A boundary that cannot say "ready" or "not ready, because" is
not a boundary.

**Independent Test**: start the server with the backend running, then with it stopped,
then with a backend announcing an unknown contract version. In each case the operator's
readiness view shows the state and reason within a few seconds of boot.

**Acceptance Scenarios**:

1. **Given** the backend is running with a known contract version, **When** the server
   boots, **Then** the boundary reports ready and consequential commands are accepted.
2. **Given** the backend is not running, **When** the server boots, **Then** the
   boundary reports not ready with reason "backend unreachable", players still enter
   the world, ordinary play (movement, combat, item movement) is available, and every
   consequential command is refused with that reason.
3. **Given** the backend announces a contract version the game does not know, **When**
   the server boots, **Then** the boundary reports not ready with reason "contract
   version unknown" and behaves as in scenario 2.
4. **Given** the boundary was not ready, **When** the backend becomes reachable with a
   known version, **Then** the boundary reports ready without a server restart.

---

### User Story 2 - A player's consequential command gets exactly one definitive answer (Priority: P1)

A player asks for something that changes a logical record. The request travels as one
typed command with an operation id, the actor, the target, and the expected revision.
The answer is one of: accepted with a receipt, refused with a stable reason and the
current authoritative value, or already applied with the original result. The same
command sent twice never applies twice.

**Why this priority**: this is the shape every future command takes. Idempotency and
stable refusals are what make retries after a timeout or crash safe.

**Independent Test**: with the wallet as the first record, a player views their balance,
an operator credits them, and the player sees the new balance and a receipt. Repeating
the credit with the same operation id changes nothing and returns the original receipt.

**Acceptance Scenarios**:

1. **Given** an audited player with a wallet, **When** they ask for their balance,
   **Then** they see total and reserved money as the backend records them.
2. **Given** an accepted command, **When** the same operation id is sent again with the
   same payload, **Then** the answer is "already applied" with the original receipt and
   no second ledger entry exists.
3. **Given** an accepted command, **When** the same operation id is sent again with a
   different payload, **Then** it is refused and a security event is recorded.
4. **Given** a command whose expected revision is stale, **When** it is sent, **Then**
   it is refused with the current revision so the caller can refresh and retry.
5. **Given** the backend does not answer within the configured time, **When** the
   player waits, **Then** they see a refusal naming the reason and nothing was applied
   on the game side; if the backend had in fact applied it, the retry returns "already
   applied".
6. **Given** a player whose identity is not yet audited, **When** they issue any
   consequential command, **Then** it is refused with a retryable "identity not ready"
   reason and no record is created for them.

---

### User Story 3 - An operator corrects a wallet and the correction is on the record (Priority: P2)

An authorized operator, from inside the game, credits or debits a player's wallet with
a reason. The player receives a receipt (after reconnect if offline). The operator's
identity, reason, before and after balance, time and outcome are appended to an audit
trail the operator cannot rewrite. Unauthorized players cannot issue the command.

**Why this priority**: it is the first cockpit command, the admin control the
constitution requires for testing wallet-dependent features, and the proof that audit
works end to end.

**Independent Test**: an operator credits a test player; the player's balance changes;
the audit view shows one entry; a non-operator attempting the same command is refused
and that attempt is also audited.

**Acceptance Scenarios**:

1. **Given** an operator and a target player, **When** the operator credits an amount
   with a reason, **Then** the wallet total rises by that amount, one ledger entry and
   one audit entry exist, and the player receives a receipt.
2. **Given** a debit larger than the available balance, **When** the operator sends it,
   **Then** it is refused with the current balance and the refusal is audited.
3. **Given** a player without the operator role, **When** they send the command,
   **Then** it is refused as unauthorized and audited as a security event.
4. **Given** the target player is offline, **When** the credit is accepted, **Then**
   the receipt is shown to them on their next connection.

---

### User Story 4 - A tester drives the boundary through its states in minutes (Priority: P3)

In the demo world, a tester can put the boundary into each state (ready, unreachable,
unknown version, slow) and watch the game respond, without editing files or restarting
the game server.

**Why this priority**: the demo world is the only acceptance environment. If the
failure paths can only be produced by real outages, they will never be tested.

**Independent Test**: the tester stops and restarts the backend, switches its announced
version, and delays its answers; the operator readiness view and the player refusals
follow each change.

**Acceptance Scenarios**:

1. **Given** a ready boundary, **When** the tester stops the backend, **Then** the next
   consequential command is refused with "backend unreachable" and the readiness view
   updates without a restart.
2. **Given** the tester makes the backend answer slower than the configured time,
   **When** a player issues a command, **Then** the player sees the timeout refusal and
   a later retry resolves to either accepted or already applied, never a duplicate.

---

### Edge Cases

- The backend answers after the game has already given up: the game must not apply
  anything from a late answer; the next retry with the same operation id resolves it.
- The backend restarts while players are connected: in-flight commands resolve by
  retry, the boundary returns to ready on its own.
- The backend is upgraded to a contract version the running game does not know: the
  boundary drops to not ready with "contract version unknown" until the game is
  restarted against a matching version.
- A player disconnects between sending a command and receiving the answer: the result
  is kept for them and shown on reconnect (RULES 17.2).
- A player's command and an operator's correction race on the same wallet: exactly one
  wins the revision; the other is refused with the new revision. A player's own second
  command never races, because it is refused as busy while the first is pending.
- A player's connect fails because the backend is away at join time: the player is
  inside the world without a record; the game repeats the connect for every such
  player the moment the boundary becomes ready again, so no rejoin is needed
  (observed 2026-09-04 when the backend was closed mid-test).
- The game receives an identity it cannot use (empty or null stable id): the player is
  treated as not audited; no record is created from a name or connection id. A
  dedicated server without backend reach and non-dedicated play both produce this, so
  the development flag of FR-020 exists; without it the feature is not testable
  locally.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The game server MUST verify at boot that the backend is reachable on the
  same machine and announces exactly the one contract version the game build knows,
  and MUST re-check on a schedule until it does, so readiness recovers without a
  restart.
- **FR-002**: The game MUST expose the boundary state (ready / not ready with a stable
  reason) to authorized operators inside the game.
- **FR-003**: Every consequential command MUST carry an operation id, the acting
  identity (and subject when different), the target's stable id, the expected revision
  where edits can race, the configuration revision, and only the choice the server
  cannot derive; cockpit commands additionally carry an operator reason.
- **FR-004**: The backend MUST answer every command with exactly one of: accepted (with
  the result and a receipt), refused (with a stable reason and the current authoritative
  value needed to correct the request), or already applied (with the original result).
- **FR-005**: The game MUST apply a physical or displayed effect only after an accepted
  answer, never before and never on a late or unknown answer.
- **FR-006**: While the boundary is not ready or the backend does not answer within the
  configured time, the game MUST refuse consequential commands with a visible, stable
  reason and MUST NOT journal them for later replay. Ordinary play is unaffected.
- **FR-007**: A repeated operation id with an identical payload MUST return the original
  result without a second application; a repeated operation id with a different payload
  MUST be refused and recorded as a security event.
- **FR-008**: The backend MUST be the system of record for the player record, the
  wallet (total, reserved, revision) and the append-only ledger (RULES 7.1). The game
  keeps no copy of money beyond what it displays.
- **FR-009**: Player records MUST be keyed by the stable identity available after the
  connection's audit success; before that every identity-keyed command is refused with
  a retryable "identity not ready" reason and nothing is created.
- **FR-010**: Players MUST be able to see their own balance (total and reserved) at the
  point of use, and only their own (visibility: current player).
- **FR-011**: Authorized operators MUST be able to credit or debit a player's wallet
  with a reason; the command is refused for unauthorized identities, for debits
  exceeding the available balance, and while the boundary is not ready (RULES 17.4,
  17.5).
- **FR-012**: Every accepted or refused consequential command MUST produce one
  append-only audit entry holding actor, role, reason, operation id, target, expected
  and actual revision, before/after references, time, outcome and correlation id, and
  authorized operators MUST be able to read the recent entries in game.
- **FR-013**: Every accepted consequential command MUST produce a receipt keyed by its
  operation id, delivered to the affected player, and held for them when offline
  (RULES 17.2).
- **FR-014**: Every user-visible string of this feature (refusal reasons, receipts,
  readiness states, operator feedback) MUST be localized in English and Russian.
- **FR-015**: The backend MUST listen only on the local machine and the game MUST call
  no other host during play.
- **FR-016**: The following MUST be adjustable by an authorized operator at runtime
  without a restart: the backend answer time limit and the readiness re-check interval.
- **FR-017**: The demo world MUST be able to produce every boundary state (ready,
  unreachable, unknown version, slow answer) using only the backend's own controls and
  the operator commands of this feature.
- **FR-018**: Operators MUST have an in-game cockpit panel that opens only for the
  operator role and shows the boundary state, a wallet lookup by player, a credit/debit
  form with a mandatory reason, and the most recent audit entries; players MUST have an
  in-game display of their own balance.
- **FR-019**: A player MUST have at most one consequential command in flight; a second
  command while one is pending is refused immediately with a stable "busy" reason and
  causes no backend call.
- **FR-020**: The game MUST offer a development-only substitute for a missing audited
  identity, switched on by a flag in the server's backend config that is off by
  default: the identity is derived from the player name exactly as the engine derives
  it for non-dedicated play, so it is stable across sessions and identical to the one
  Workbench play would give. Clients of one platform account share a name, so a
  joiner whose derived identity is already held by a connected or connecting session
  is derived from the name with a join counter ("Name#2", "Name#3"); join order
  decides which client is which. With the flag on the server MUST log a warning at
  start and at every substitution; with the flag off FR-009 applies unchanged.

### Key Entities

- **Boundary state**: ready or not ready, with a stable reason and the time last
  checked; visible to operators.
- **Contract version**: the version the backend announces and the single version the
  game build knows; anything but an exact match is unknown.
- **Command**: operation id, actor, subject, target stable id, expected revision,
  configuration revision, operator reason, and the single player choice.
- **Answer**: accepted, refused (reason, current value), or already applied (original
  result).
- **Player record**: stable identity, creation time; the anchor for every later
  per-player record.
- **Wallet**: owner, total, reserved, revision.
- **Ledger entry**: wallet, operation id, reason type, signed amount, resulting balance,
  actor, time; append-only.
- **Receipt**: operation id, affected player, what happened, delivered flag.
- **Audit entry**: actor, role, reason, operation id, target, revisions, before/after
  references, time, outcome, correlation id; append-only.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: With the backend running, a balance query or an operator credit answers
  the player within 1 second in the demo world.
- **SC-002**: With the backend stopped, 100% of consequential commands are refused with
  a visible reason within the configured time limit, and players keep moving, fighting
  and moving items throughout.
- **SC-003**: Across a drill of 100 retried commands (timeouts, backend restarts, game
  crashes between send and answer), the ledger holds exactly one entry per operation
  id and wallet totals equal the sum of their ledger.
- **SC-004**: A backend announcing an unknown contract version yields zero accepted
  consequential commands and one operator-visible reason within one re-check interval.
- **SC-005**: Every consequential command, accepted or refused, has exactly one audit
  entry an operator can find by operation id.
- **SC-006**: A receipt for a command accepted while its player was offline is shown
  within 10 seconds of their next connection.
- **SC-007**: Every user-visible string of the feature is present in both English and
  Russian before the feature is played in the demo world.

## Assumptions

- The operator role for this feature is a configured allowlist of stable identities;
  the engine's role service is a later spike and, when proven, replaces the allowlist
  without changing the command.
- Identity, wallet and ledger are the only logical records the backend holds after this
  feature; faction, vehicles, plots, auction, weekly results and the wipe are later
  features.
- The wallet has no money source or sink other than operator compensation until the
  market and recovery features exist; balances therefore start at zero.
- The cockpit panel of this feature is the first slice of the launch cockpit (RULES
  17.3): it opens only for the operator role and later features add their views to it
  rather than building a second one.
- The demo world is a local dedicated server with the backend beside it. That server
  has no backend reach, so testers hold development identities (FR-020) that are
  stable across sessions; a real server audits real identities and the flag stays off.
  With the flag on, Workbench play exercises the feature too.
- No website, RCON namespace, host supervisor, backup or persistence spike is in scope.
- The backend keeps its records durably across its own restarts; how is its concern,
  not the game's.

## Source Revisions

The constitution moves the system of record for logical records to the local backend,
and `TECHNICAL-DESIGN.md` v1.0 said otherwise. The operator accepted the following
revisions on 2026-09-03 and they are applied as `TECHNICAL-DESIGN.md` v1.1:

- **2.5 Chosen launch topology**: item 2 becomes two items — the native persistence
  system stores the physical world and linking ids only; the local backend stores every
  logical record. The last paragraph's "None is a second source of truth" keeps its
  meaning for the website and exports; the backend is the first source of truth, not a
  second.
- **4 Persistence, Transactions, and Recovery**: the opening paragraph, 4.1 and 4.2
  are restated for two stores. The durable-record table splits into backend records
  (everything listed except `ReconnectState` and the native storage roots) and
  physical-world records. The strict whole-world save batch in 4.2 is replaced by "one
  backend call per consequential command, physical effect after acceptance"; the
  checkpoint coordinator keeps its role for the physical world only. 4.3 boot adds
  "contract version check and boundary readiness" as step 1 and reconciles the
  physical world against backend records by operation id.
- **16.3 Remote bridge and host supervisor**: the last paragraph's "may never mutate a
  database behind the running game server" is restated as "the website synchronises
  with the backend, never with the game", matching the constitution.
- **16.4 Audit and security events**: the audit trail lives in the backend; the
  `$profile` spool paragraph is removed.
- **18.2 Disconnect and persistence**: the bullet "Add Klondike's custom
  `PersistentState` to this model instead of a parallel JSON store" is narrowed to the
  physical world; the backend note in 18.6 ("optional for export") is replaced by the
  runtime client being the launch path for logical records.
- **19 Required Technical Spikes**: spike 4 is retargeted from whole-world save
  batching to backend call durability and rate; spike 11 keeps the native resume for
  the physical world only.

No `RULES.md` change is needed: player-observable behaviour (7.1, 17.2, 17.4, 17.5) is
unchanged.
