# Contract: Game ↔ Local Backend (HTTP, contract version "1")

Base URL from `$profile:TK_Backend.json`, default `http://127.0.0.1:8471/`. The backend
binds `127.0.0.1` only. All bodies are JSON, all times UTC RFC 3339. The game sets
`RestContext.SetTimeout(answer_timeout_s)`.

Transport rule: any HTTP status other than 200, a timeout, or an unparsable body means
"the boundary did not answer"; the game refuses the player's command with
`backend_unreachable` and applies nothing. Domain outcomes are always HTTP 200.

## GET /v1/health

Response 200:

```json
{
  "contract": "1",
  "config_revision": 3,
  "answer_timeout_s": 5,
  "recheck_interval_s": 15,
  "time": "2026-09-03T18:00:00Z"
}
```

The game compares `contract` with its constant; a difference is `VERSION_UNKNOWN`.
`answer_timeout_s` and `recheck_interval_s` are applied immediately.

## POST /v1/players/{uuid}/connect

Request:

```json
{ "display_name": "Ivanov" }
```

Response 200 (creates the player and a zero wallet when absent; marks the returned
receipts delivered in the same transaction):

```json
{
  "player":   { "uuid": "…", "role": "operator", "created_at": "…" },
  "wallet":   { "total": 1200, "reserved": 0, "revision": 4 },
  "receipts": [
    { "op_id": "…", "kind": "compensation", "amount": 200, "total_after": 1200,
      "reason": "missing payout 2026-09-01", "created_at": "…" }
  ]
}
```

## GET /v1/players/{uuid}/wallet

Response 200: `{ "total": 1200, "reserved": 0, "revision": 4 }`.
Response 200 with `{ "error": "player_unknown" }` when no record exists.

## POST /v1/commands

Request envelope. Required for every type: `op_id`, `type`, `actor`, `subject`,
`target`, `expected_revision`, `config_revision`, `payload`; `subject` equals `actor`
unless acting for someone else. Required for cockpit types: `reason`. Optional, for
`wallet.compensate` only: `target_online`. The backend and the game validate the same
list.

```json
{
  "op_id": "5b1f…",
  "type": "wallet.compensate",
  "actor": "<operator uuid>",
  "subject": "<operator uuid>",
  "target": "<player uuid>",
  "expected_revision": 4,
  "config_revision": 3,
  "reason": "missing payout 2026-09-01",
  "payload": { "amount": 200 },
  "target_online": true
}
```

`target_online` (optional, `wallet.compensate` only) says the target has a live
session on the game server. The game then pushes the receipt at once, so the backend
records it as delivered at creation and the next connect does not replay it. It sits
outside the payload so a retry that finds the target gone is still the same
operation.

Types and payloads:

| type | payload | target | who |
|---|---|---|---|
| `wallet.compensate` | `{ "amount": <signed int, non-zero> }` | player uuid | operator |
| `config.set` | `{ "answer_timeout_s": 1..120, "recheck_interval_s": 1..600 }` (one or both) | `config` | operator |
| `security` | `{ "kind": "unauthorized_cockpit_rpc", "detail": "…" }` | actor uuid | game server, actor = the offending player |

Response 200:

```json
{
  "status": "accepted",
  "reason_code": "",
  "revision": 5,
  "wallet": { "total": 1400, "reserved": 0, "revision": 5 },
  "receipt": { "op_id": "5b1f…", "kind": "compensation", "amount": 200,
               "total_after": 1400, "reason": "…", "created_at": "…" }
}
```

`status` is one of `accepted`, `refused`, `already_applied`. On `refused`, `wallet`
carries the current authoritative values so the caller can correct and retry; on
`already_applied`, the whole body is the recorded original answer. `wallet` and
`receipt` are absent for `config.set`, which returns `revision` = the new
configuration revision instead.

Reason codes (stable, localized by the client as `TK-Reason_<code>`):

| code | meaning |
|---|---|
| `backend_unreachable` | no answer within the limit, or a transport error (game-side) |
| `contract_version_unknown` | boundary not ready because of the version (game-side) |
| `identity_not_ready` | the actor has no audited identity yet (game-side) |
| `busy` | the actor already has a command pending (game-side) |
| `target_ambiguous` | a name held by several connected players (game-side) |
| `unauthorized` | actor is not an operator |
| `player_unknown` | target has no player record, or the command named no target (game-side) |
| `invalid_amount` | zero, or the amount or the resulting total outside the signed 32-bit range the game can represent |
| `insufficient_funds` | debit larger than `total - reserved` |
| `stale_revision` | `expected_revision` differs from the wallet's |
| `reason_required` | empty reason on a cockpit command |
| `op_id_payload_mismatch` | same `op_id` with a different payload, type, actor, subject or target; audited as security |
| `invalid_config_value` | out of the allowed range |

## GET /v1/audit?limit=20&before=<id>

Response 200:

```json
{ "entries": [
  { "id": 41, "created_at": "…", "op_id": "…", "actor_uuid": "…", "actor_role": "operator",
    "type": "wallet.compensate", "target": "…", "outcome": "accepted",
    "reason_code": "", "reason": "…", "amount": 200 }
] }
```

Newest first; `before` pages backwards. `limit` is capped at 50. Only the game calls
this; the game forwards rows only to operators.

## Backend runtime flags (demo-world controls)

| flag | effect |
|---|---|
| `-listen 127.0.0.1:8471` | listen address; refuses to start on a non-loopback address |
| `-data <dir>` | database and seed file directory |
| `-announce-contract <string>` | announce this contract version in `/v1/health` |
| `-delay <duration>` | delay every `POST /v1/commands` answer by this much (for example `7s`); health and reads stay instant |

Seed file `klondiked.json` in the data directory, read at every start. The operator
list is an allowlist applied each start (listed identities become operators, created
if unknown; every other operator becomes a player); the runtime values are taken only
when the database is new, because afterwards they carry a revision:

```json
{ "operators": ["<uuid>", "<uuid>"], "answer_timeout_s": 5, "recheck_interval_s": 15 }
```
