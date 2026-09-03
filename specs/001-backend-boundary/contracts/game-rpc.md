# Contract: Client ↔ Game Server (owner RPCs on `TK_PlayerComponent`)

Every request is `[RplRpc(RplChannel.Reliable, RplRcver.Server)]` and every answer is
`[RplRpc(RplChannel.Reliable, RplRcver.Owner)]` to the requesting client only. No value
of this feature is replicated by property or broadcast. All RPCs have at most 8
arguments. `status` is `0 accepted, 1 refused, 2 already_applied`; `state` is
`0 NOT_CHECKED, 1 READY, 2 UNREACHABLE, 3 VERSION_UNKNOWN`; `role` is
`0 player, 1 operator`. Strings that reach the screen are reason codes, translated by
the client.

Server-side checks before any backend call, in this order: identity ready (else
`identity_not_ready`), role for cockpit RPCs (else `unauthorized`, and a `security`
command is sent to the backend), one in flight per player (else `busy`), boundary ready
(else the boundary's reason code).

| Request (client → server) | Answer (server → owner) | Who |
|---|---|---|
| `RpcAsk_OpenCockpit()` | `RpcDo_OwnerCockpitState(int state, string reasonCode, string contractSeen, int configRevision, int answerTimeoutS, int recheckIntervalS, int role)` | anyone; non-operators get `state`/`role` only meaningfully |
| `RpcAsk_GetWallet(string targetUuid)` (empty = self) | `RpcDo_OwnerWallet(string targetUuid, int total, int reserved, int revision, string reasonCode)` | self: anyone; other: operator |
| `RpcAsk_Compensate(string opId, string targetUuid, int amount, int expectedRevision, string reason)` | `RpcDo_OwnerCommandResult(string opId, int status, string reasonCode, int total, int reserved, int revision)` | operator |
| `RpcAsk_SetBoundaryConfig(string opId, int answerTimeoutS, int recheckIntervalS, string reason)` | `RpcDo_OwnerCommandResult(opId, status, reasonCode, 0, 0, configRevision)` | operator |
| `RpcAsk_GetAudit(int beforeId)` | `RpcDo_OwnerAuditBegin()`, then up to 20 × `RpcDo_OwnerAuditRow(int id, string time, string actorUuid, string type, string target, string outcome, string reasonCode, int amount)`, then `RpcDo_OwnerAuditEnd(int nextBeforeId)` | operator |
| (none; pushed) | `RpcDo_OwnerReceipt(string opId, int kind, int amount, int totalAfter, string time, string reason)` | the affected player, on connect (claimed receipts) and live on acceptance |

Rules:

- The client mints `opId` with `UUID.GenV4()` when the operator presses the button and
  keeps it until the result arrives; a retry after `backend_unreachable` reuses it.
- `targetUuid` may also be the exact display name of a connected player; the server
  resolves it to that player's identity before any check, because names are what an
  operator can read off the player list at a test table.
- `expectedRevision` is the revision the panel last displayed for that wallet; the
  server passes it through unchanged.
- `RpcDo_OwnerReceipt` carries the new `totalAfter`, so the target's display updates
  from the receipt without a second query.
- `RpcAsk_GetAudit` results are transient; the panel does not cache pages.
- A player with a pending command who disconnects loses the slot; the backend still
  resolves the command, and the receipt (if any) is claimed on the next connect.
