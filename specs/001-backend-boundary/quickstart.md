# Quickstart: Backend Boundary

How to prove the feature in the demo world. Contracts: [backend-http.md](contracts/backend-http.md),
[game-rpc.md](contracts/game-rpc.md). Records: [data-model.md](data-model.md).

## Prerequisites

- Go toolchain on the machine that builds the backend (not installed on the
  development machine today).
- A local dedicated Arma Reforger server running the `KlondikeDemo` scenario, plus one
  or two clients. Designed setup: the addon loaded from disk (`-addonsDir` naming the
  directory that contains `Triad Klondike`, `-addons TriadKlondike`), profile
  directory `dsprofile`, `game.scenarioId` = `{<GUID of Missions/KlondikeDemo.conf.meta>}Missions/KlondikeDemo.conf`,
  `maxPlayers` at least 2, `visible` false and BattlEye off (unsigned addon),
  `-maxFPS 60`. Two identities are needed: one operator, one player; two clients on
  one machine share the platform name and get numbered development identities.
- `$profile:TK_Backend.json` on the game server:
  `{ "BaseUrl": "http://127.0.0.1:8471/", "DevIdentityFromName": true }`. The local
  server has no backend reach, so without the flag every player audits with an empty
  identity (FR-020). The flag stays off on a real server.
- Backend data directory with `klondiked.json` listing the operator's uuid. With the
  flag on, the server log prints the derived id at connect
  (`using development identity 00bbbddd-…`); copy it from there.

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

Result 2026-09-04, local dedicated server with two clients (development identities):
steps 1–3 passed with 30000 instead of 200. Operator saw `accepted`, revision 2, total
30000; the player saw the receipt ("Wallet changed by 30000, now 30000: Present")
and the new balance at once; the audit list showed the accepted row plus the two
`stale_revision` refusals from before the panel fix (the panel wiped the returned
revision when no lookup preceded Apply; fixed the same day). Step 4 passed the same
session: a debit of 30500 against 30000 was refused `insufficient_funds` at revision 2
and the panel showed "not enough money in the wallet". Step 5 has no path from an
unmodified client (the operator section never renders for a player); the refusal is
covered by the backend test for `unauthorized` and the server-side role gate.

## Scenario C: idempotency and retries (US2, US4 / SC-003)

1. Start the backend with `-delay 7s` while the answer limit is 5 s.
2. Operator credits 100. Expected: `backend_unreachable` after the limit, panel keeps
   the operation id, nothing applied on screen.
3. Operator presses retry (same operation id) after the delay flag is removed.
   Expected: exactly one ledger row for the operation id. The answer is
   `already_applied` with the original receipt when the first call reached the
   backend late, or `accepted` when the engine gave up before sending it; both
   satisfy SC-003.
4. Repeat the credit with the same operation id but a different amount from a
   modified client. Expected: `op_id_payload_mismatch`, one security audit row.
5. Kill the game server between send and answer (backend on `-delay 4s`, under the
   answer limit), restart, reconnect. Expected: the ledger has exactly one row for the
   operation id and the wallet its new revision. The receipt was created as delivered
   because the target was online at send time and the server died before pushing it,
   so the rejoin shows `receipts=0`; the notification for that credit is lost, the
   money and the ledger are not. Accepted by the operator on 2026-09-04 as the price
   of settling delivery without a second call per compensation. A crash before
   the backend has read the request loses the command entirely (connection reset
   discards the buffered body): nothing applied, nothing to retry, which is the
   fail-closed outcome. The delay flag therefore sleeps after reading the body.

Result 2026-09-04 for step 5, three runs on `-delay 4s`: (a) crash before the
backend had read the request: nothing applied, `body not received` logged; (b) crash
after the read, no prior lookup: `refused stale_revision`, one operation row; (c)
crash after the read with a lookup: `accepted revision=12` during the outage, both
rejoins `receipts=0`, balance 164000, exactly one row. Step 4 (modified client) not
played; covered by the backend test for `op_id_payload_mismatch`.

Result 2026-09-04, steps 1–2 with `-Delay 7s` against a 5 s limit: passed. Header
turned unreachable, the credit answered "the backend did not answer", the panel
kept the operation. Observation: the backend never received the command (a probe
proved the backend processes a client-aborted request after the delay; the
command simply was not sent), so the engine drops a request whose timeout expires
before it is dispatched. Step 3 therefore answers `accepted`, not
`already_applied`; expectation amended. Step 3 played after the restart without
delay: `accepted` at revision 5 with the kept operation id, one ledger row, receipt
and balance on the player client, one command line on the backend console. The
`invalid_amount` audit row of 20:57:53 is the agent's probe, not a game action.
Steps 4–5 not played (step 4 needs a modified client; step 5 a server kill).

## Scenario D: busy and identity (FR-009, FR-019)

1. With `-delay 3s`, send two credits back to back. Expected: the second is `busy`
   immediately and causes no backend call (backend log shows one command).

Result 2026-09-04 for step 1, local dedicated server: passed on the server side. A
single press was accepted after 3 s (revision 6); two quick presses produced one
command line and one acceptance (revision 7), the second press answered `busy`. Panel
bug found: the second press replaced the pending operation, so the acceptance of the
first press was dropped as unknown and the operator saw only the busy text; fixed the
same day by refusing a press locally while an answer is outstanding. Re-test with the
fix (delay on commands only, 5000 credit): busy shown at once, replaced by
`accepted` at revision 8 after 3 s, one command line, receipt delivered. The `-delay`
flag was narrowed to commands the same night: a delay on every answer made the
engine's serialised request queue time out after a join burst.
2. Join on a non-dedicated session with `DevIdentityFromName` off (or absent).
   Expected: every identity-keyed RPC answers `identity_not_ready`; no player record
   is created.
3. Set `DevIdentityFromName` to true and join again (Workbench or the local server).
   Expected: the server log warns at start and prints `using development identity
   00bbbddd-…` at connect; the same name gets the same id on every join; the balance
   line shows a wallet. A second client under the same platform name gets a different
   id (derived from `Name#2`), so two clients on one machine hold two wallets; the
   first joiner is the operator when the seed lists the plain-name id.

Result 2026-09-04 for step 3, local dedicated server: passed. Two clients under one
platform name got `…fc7c0002557e` (operator, first joiner) and `…fc7c0c7b340f`
(player), stable across four joins; the backend console showed both connects with
their roles.

Result 2026-09-03 for step 2, from Workbench: passed. The balance line showed the
identity refusal in every boundary state, and the server never called connect. Step 1
needs the dedicated server.

## Scenario D2: join during an outage

1. Stop the backend (`tools/backend.ps1 -Stop`), join with a client. Expected: the
   server log shows `connect for player N failed`, the balance line says the identity
   is not confirmed.
2. Start the backend again. Expected: within one re-check interval the state turns
   ready, the server log shows a second connect for that player, Refresh shows the
   wallet. No rejoin.

Receipt delivery (T046, 2026-09-04): passed. A credit to the online player, then a
rejoin: `receipts=0`. A credit by identity while the player was disconnected, then a
rejoin: `receipts=1`, balance updated. A credit by NAME while the player was away
landed on the operator, because the name resolves over connected sessions only and
both clients share the name; offline targets need the identity. The receipt pushed
at connect is not shown by a panel opened afterwards (panel keeps no receipt
history); redo scope.

Result 2026-09-04, replayed with the retry: passed. Both clients joined with the
backend stopped (connects failed with HTTP 0, panels unreachable with identity not
confirmed); after the backend start the server log showed `UNREACHABLE -> READY`
and a new connect per player with the same development identities, the backend
console showed both connects, and the panels showed the wallets on Refresh. No
rejoin. Observation: the player's connect claimed one receipt that had already been
delivered live before the outage; receipts pushed live are never marked delivered,
so every reconnect re-delivers them (six after the earlier rejoin). Open question
for the operator, see the knowledge note.

## Scenario E: runtime regulation (FR-016)

1. From the panel, set the answer limit to 2 s and the interval to 5 s with a reason.
2. Expected: `accepted`, configuration revision +1, one audit row, the panel's values
   update within one interval, and Scenario C now times out after 2 s.

Result 2026-09-04, local dedicated server: passed for two consecutive changes
(answer limit 10 s then interval 10 s): `accepted` with configuration revisions 2 and
3, one `config.set` audit row each, the panel header showed the new values at the
next poll. The timeout half (Scenario C after 2 s) was not played. A panel opened
before the change keeps the old header until Refresh, by design (D4: values travel
with the health answer).

## Scenario F: language (SC-007)

Switch the client to Russian and repeat A and B: every state, reason and receipt is in
Russian; no English or raw code appears.
