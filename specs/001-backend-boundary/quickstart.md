# Quickstart: Backend Boundary

How to prove the feature in the demo world. Contracts: [backend-http.md](contracts/backend-http.md),
[game-rpc.md](contracts/game-rpc.md). Records: [data-model.md](data-model.md).

## Prerequisites

- Go toolchain on the machine that builds the backend (not installed on the
  development machine today).
- A local dedicated Arma Reforger server (identities are audited only there) running
  the `KlondikeDemo` scenario (`Missions/KlondikeDemo.conf`, Arland, US and USSR
  spawn points, deploy menu), plus one or two clients. Two identities are needed:
  one operator, one player.
- `$profile:TK_Backend.json` on the game server: `{ "BaseUrl": "http://127.0.0.1:8471/" }`.
- Backend data directory with `klondiked.json` listing the operator's uuid.

## Build and run the backend

```text
cd backend
go build ./cmd/klondiked
go test ./...
./klondiked -listen 127.0.0.1:8471 -data ./data
```

Expected: `go test` passes (idempotency, revision, insufficient funds, payload
mismatch, config range); the process logs the contract version and listen address and
refuses any non-loopback `-listen`.

## Scenario A: boot against a running backend (US1 / SC-004)

1. Start the backend, then the game server, connect the operator, press the cockpit key.
2. Expected: state READY, contract "1", configuration revision, both runtime values.
3. Stop the backend. Within one re-check interval the panel's refresh shows
   UNREACHABLE with `backend_unreachable`.
4. Start it with `-announce-contract 2`. Refresh shows VERSION_UNKNOWN.
5. Restart it normally. Refresh shows READY without touching the game server.

Result 2026-09-03, played from Workbench (host is the server): passed. Unreachable at
boot with no backend, ready after the backend started, "unknown contract version" with
`-announce-contract 2`, ready again after the normal restart; the game was never
restarted. Contract and configuration values displayed as announced.

## Scenario B: balance, credit, receipt (US2, US3 / SC-001, SC-005)

1. Player connects and opens the panel: balance 0 / reserved 0.
2. Operator looks up the player's wallet, credits 200 with a reason.
3. Expected within a second: operator sees `accepted`, revision 2, total 200; the
   player sees a receipt and total 200; the audit page shows one row with the
   operation id; `go`'s SQLite file has one ledger row.
4. Operator debits 500. Expected: `insufficient_funds` with the current total; one
   refused audit row.
5. Player (non-operator) forces the credit RPC (test build hook or a second client
   without the role). Expected: `unauthorized`, one `security` audit row.

## Scenario C: idempotency and retries (US2, US4 / SC-003)

1. Start the backend with `-delay 7s` while the answer limit is 5 s.
2. Operator credits 100. Expected: `backend_unreachable` after the limit, panel keeps
   the operation id, nothing applied on screen.
3. Operator presses retry (same operation id) after the delay flag is removed.
   Expected: `already_applied` with the original receipt, exactly one ledger row.
4. Repeat the credit with the same operation id but a different amount from a
   modified client. Expected: `op_id_payload_mismatch`, one security audit row.
5. Kill the game server between send and answer, restart, reconnect: the player's
   receipt arrives on connect; the ledger still has exactly one row per operation id.

## Scenario D: busy and identity (FR-009, FR-019)

1. With `-delay 3s`, send two credits back to back. Expected: the second is `busy`
   immediately and causes no backend call (backend log shows one command).
2. Join on a non-dedicated session. Expected: every identity-keyed RPC answers
   `identity_not_ready`; no player record is created.

Result 2026-09-03 for step 2, from Workbench: passed. The balance line showed the
identity refusal in every boundary state, and the server never called connect. Step 1
needs the dedicated server.

## Scenario E: runtime regulation (FR-016)

1. From the panel, set the answer limit to 2 s and the interval to 5 s with a reason.
2. Expected: `accepted`, configuration revision +1, one audit row, the panel's values
   update within one interval, and Scenario C now times out after 2 s.

## Scenario F: language (SC-007)

Switch the client to Russian and repeat A and B: every state, reason and receipt is in
Russian; no English or raw code appears.
