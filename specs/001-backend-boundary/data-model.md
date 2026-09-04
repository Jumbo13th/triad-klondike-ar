# Data Model: Backend Boundary

All records below live in the local backend (SQLite). The game keeps only transient
per-connection state. Every table has an integer primary key unless stated, and every
timestamp is UTC RFC 3339 text.

## Backend records

### players

| Field | Type | Rule |
|---|---|---|
| uuid | text, unique | the audited stable identity; never a name or connection id |
| role | text | `player` or `operator`; set from the seed's operator list at every backend start |
| display_name | text | last name seen at connect; informational only |
| created_at | text | first connect |
| last_connect_at | text | updated on every connect |

### wallets

| Field | Type | Rule |
|---|---|---|
| player_uuid | text, unique, references players | one wallet per player |
| total | integer | never negative |
| reserved | integer | never negative, never greater than total; always 0 in this feature |
| revision | integer | starts at 1, +1 per accepted mutation |

Invariant checked on every commit: `0 <= reserved <= total`.

### operations

| Field | Type | Rule |
|---|---|---|
| op_id | text, unique | minted by the client (`UUID.GenV4()`), bound here |
| type | text | `wallet.compensate`, `config.set`, `security` |
| actor_uuid | text | who issued it |
| subject_uuid | text | who it is done as, when different; else equal to actor |
| target | text | wallet player uuid, or `config` |
| payload_hash | text | SHA-256 of the canonical payload |
| expected_revision | integer | as sent |
| config_revision | integer | as sent |
| reason | text | operator reason; required for cockpit commands |
| status | text | `accepted`, `refused` |
| reason_code | text | empty when accepted |
| result_json | text | the exact answer returned, replayed on repeat |
| created_at | text | |

Idempotency: a `POST /v1/commands` whose `op_id` exists replays `result_json` when the
payload hash, type, actor, subject and target all match: status `already_applied` when
the recorded operation was accepted, the recorded refusal unchanged when it was
refused. Any difference is refused `op_id_payload_mismatch` with a `security` audit
entry.

### ledger

| Field | Type | Rule |
|---|---|---|
| id | integer | append order |
| player_uuid | text | |
| op_id | text, references operations | one entry per accepted wallet operation |
| kind | text | `compensation` |
| amount | integer | signed |
| total_after | integer | wallet total after this entry |
| actor_uuid | text | operator |
| reason | text | |
| created_at | text | |

Invariant: for every player, `wallets.total` equals the sum of `ledger.amount`.

### receipts

| Field | Type | Rule |
|---|---|---|
| id | integer | |
| op_id | text, unique | one receipt per accepted consequential command |
| player_uuid | text | the affected player |
| kind | text | `compensation` |
| amount | integer | signed |
| total_after | integer | |
| reason | text | |
| created_at | text | |
| delivered_at | text, nullable | set by the connect claim or the live push |

### audit

| Field | Type | Rule |
|---|---|---|
| id | integer | append order; never updated or deleted |
| op_id | text | correlation id; empty for backend-internal events |
| actor_uuid | text | |
| actor_role | text | |
| type | text | operation type or `security`, `config`, `connect` |
| target | text | |
| reason | text | |
| expected_revision | integer | |
| actual_revision | integer | |
| before_json | text | the affected record before |
| after_json | text | the affected record after |
| outcome | text | `accepted`, `refused`, `security` |
| reason_code | text | |
| amount | integer | signed wallet change of an accepted compensation; 0 otherwise |
| created_at | text | |

### config

| Field | Type | Rule |
|---|---|---|
| revision | integer, primary key | +1 per accepted `config.set` |
| answer_timeout_s | integer | 1..120 |
| recheck_interval_s | integer | 1..600 |
| changed_by | text | actor uuid or `seed` |
| reason | text | |
| created_at | text | |

The newest row is the effective configuration; older rows are the history the design
requires (TECHNICAL-DESIGN 3.2).

## Game-side transient state (server only)

| Where | State | Lifetime |
|---|---|---|
| `TK_BackendComponent` | boundary state, last check time, contract version seen, config revision, answer timeout, re-check interval | process |
| `TK_BackendComponent` | map player id → `{uuid, role}` | connect to disconnect |
| `TK_BackendComponent` | map player id → pending op_id | one command |
| `TK_BackendComponent` | in-flight request holders (strong refs to callbacks) | one request |

Nothing here is replicated. The client keeps only what the panel currently shows.

## State transitions

Boundary: `NOT_CHECKED → (READY | UNREACHABLE | VERSION_UNKNOWN)`, re-evaluated on every
health answer; any state can move to any other.

Command (game side): `idle → pending → idle`, where `pending` ends on answer, timeout,
or the player's disconnect.

Receipt: `undelivered → delivered` once, by claim on connect or by live push.
