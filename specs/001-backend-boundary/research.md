# Research: Backend Boundary

Every decision below names its source. Engine facts come from the pinned game scripts
(`Arma-Reforger-Script-Diff/`, 1.8.0.10), the official REST and JSON docs
(`advanced-docs/`), and the shipped lobby addon (`lite-lobby-ar/`).

## Engine facts that shape the design

- **Outbound HTTP only.** `RestApi.GetContext(url)` returns a `RestContext` with async
  `GET/POST/PUT/DELETE(RestCallback, request, data)`, `SetTimeout(int seconds)` and
  `SetHeaders` (`GameLib/generated/online/RestContext.c`). The official doc limits data
  to under 1 MB and says custom headers are unsupported at runtime. There is no inbound
  listener in script. The game is always the caller.
- **Callback lifetime.** A `RestCallback` not held by a strong reference is deleted
  after execution (`RestCallback.c` warning). The lobby addon keeps one holder object
  per in-flight request (`LL_PlayerVerification.c`, `LL_PlayerVerificationRequest`).
- **Both callbacks may carry an HTTP code.** A 4xx can arrive through `OnError` with
  `GetHttpCode()` valid; a timeout arrives with `ERestResult.EREST_ERROR_TIMEOUT`.
  Judge by HTTP code first, then by rest result (same file, `OnVerificationError`).
- **JSON.** `JsonApiStruct` (`GameLib/generated/online/JsonApiStruct.c`): `RegV` in the
  constructor, `ExpandFromRAW(string)` to parse (no error return; unknown fields are
  ignored), `Pack()` then `AsString()` to encode (`AsString()` alone yields `{}`),
  `PackToFile` to write a fresh file. Registered names are case-sensitive.
- **Identity.** `SCR_BaseGameModeComponent.OnPlayerAuditSuccess(int playerId)` is
  dispatched by `SCR_BaseGameMode` to every game-mode component; it is the first moment
  `GetGame().GetBackendApi().GetPlayerIdentityId(playerId)` is reliable, and only on the
  server (`SCR_PlayerIdentityUtils.c`). On a non-dedicated session the vanilla utility
  synthesises an id from the player name (`00bbbddd-` prefix, three hashed name
  slices); on a dedicated server without backend reach it returns empty and reports the
  server as misconfigured. Klondike reads the backend id directly and treats empty as
  not ready; with `DevIdentityFromName` on (2026-09-04) it applies the same derivation
  itself, since the vanilla one is limited to non-dedicated sessions.
- **Operation ids.** `UUID.GenV4()` (`Core/generated/Types/UUID.c`) generates a random
  v4 id on any machine, so the client can mint the operation id as the design requires.
- **Owner RPC channel.** A `ScriptComponent` on the PlayerController is the only entity
  the client owns; `[RplRpc(RplChannel.Reliable, RplRcver.Server)] RpcAsk_*` carries a
  request up and `[RplRpc(RplChannel.Reliable, RplRcver.Owner)] RpcDo_Owner*` returns a
  result to that client alone (`LL_LobbyPlayerComponent.c`). At most 8 arguments per
  RPC; longer lists go as begin/row/end triples.
- **Menus.** A panel with a mouse cursor is a `MenuBase` registered as a
  `MenuPreset` in an override of `Configs/System/chimeraMenus.conf` carrying the base
  game's GUID `{C747AFB6B750CE9A}`, opened with `MenuManager.OpenMenu(preset)`; its
  input context is added to an override of `chimeraInputCommon.conf`; the key is an
  `Action` there and handled with `InputManager.AddActionListener` on non-dedicated
  sessions (`LL_StatsAdminPanel.c`, `LL_GameModeCoop.c`, both conf overrides).
- **Localization.** One `.st` source with `Target_en_us` and `Target_ru_ru`, thirteen
  generated runtime confs registered in `addon.gproj`; server-to-client messages send
  the key and the client translates (`lite-lobby-ar/LOCALIZATION.md`).
- **Periodic work.** `GetGame().GetCallqueue().CallLater(method, delayMs, repeat)`,
  as used throughout the game scripts, drives the readiness re-check.

## Decisions

### D1. Transport and encoding

Decision: HTTP/1.1 over `http://127.0.0.1:<port>/`, JSON bodies, one `RestContext`
created at boot, `SetTimeout` set from the configured answer-time limit. Queries use
`GET`, commands use `POST`. The backend answers every well-formed command with HTTP
200 and a `status` field; HTTP errors mean the boundary is not answering.

Rationale: the only runtime client the engine has. Separating transport failure (HTTP
error, timeout) from domain outcome (accepted, refused, already applied) keeps
fail-closed handling in one place.

Alternatives: query-string secret as the lobby addon uses for its website. Rejected:
the constitution's boundary is the loopback interface; a secret adds nothing against a
process already on the host and complicates every URL.

### D2. Backend runtime and storage

Decision: Go, standard library `net/http` and `encoding/json`, SQLite through
`modernc.org/sqlite` (pure Go, no C toolchain), one database file, WAL mode,
`synchronous=FULL`, one transaction per command.

Rationale: the constitution names Go. SQLite gives atomic multi-row commits (operation,
wallet, ledger, receipt, audit in one transaction) and a unique index on the operation
id, which is the whole idempotency mechanism. A pure-Go driver keeps the build a single
`go build` on the host.

Alternatives: bbolt (no SQL, harder audit queries); JSON files with fsync (no atomic
multi-record commit). Both rejected for durability reasons the constitution states.

Confirmed by the operator on 2026-09-03. Go is not installed on the development
machine.

### D3. Contract version

Decision: the game build carries one constant contract version string; `GET /v1/health`
returns the backend's. Any difference is `contract_version_unknown`. Version "1" for
this feature.

Rationale: clarification 2 (exact match).

### D4. Readiness

Decision: `TK_BackendComponent` on the game mode polls `GET /v1/health` at boot and on
the configured re-check interval, always (not only while down), so a backend that dies
between player commands is noticed within one interval. State is one of
`NOT_CHECKED`, `READY`, `UNREACHABLE`, `VERSION_UNKNOWN`, with the time of the last
check. The health answer also carries the configuration revision and the two runtime
values (answer-time limit, re-check interval), which the component applies at once.

Rationale: FR-001, FR-002, FR-016. Carrying the values in the health answer means one
poll serves readiness and configuration, and a cockpit change takes effect within one
interval without a restart.

Alternatives: a separate `/v1/config` endpoint. Rejected as a second poll for the same
purpose.

### D5. Command envelope and outcomes

Decision: `POST /v1/commands` with `{op_id, type, actor, subject, target,
expected_revision, config_revision, reason, payload}`. The answer is
`{status: accepted | refused | already_applied, reason_code, revision, wallet,
receipt}`. The operation id is minted by the requesting client with `UUID.GenV4()` and
bound on the backend to actor, type and a payload hash. A repeat with the same hash
returns the recorded answer; a repeat with a different hash is refused
`op_id_payload_mismatch` and audited as a security event.

Rationale: TECHNICAL-DESIGN 2.1 and 4.2, FR-003, FR-004, FR-007.

### D6. Player connect

Decision: on `OnPlayerAuditSuccess` the server reads the identity, refuses to proceed
if it is empty (unless the development flag substitutes a name-derived id, FR-020),
then `POST /v1/players/{uuid}/connect`. The backend creates the player
record and a zero wallet if absent (idempotent), returns the role, the wallet and every
undelivered receipt, and marks those receipts delivered in the same transaction. The
game caches `{uuid, role}` per player id and pushes the receipts to the owner.

Rationale: FR-008, FR-009, FR-013, SC-006 in one call per connection. Marking on claim
is acceptable because the owner RPC channel is reliable and the receipt also stays in
the ledger the operator can read.

Alternatives: a separate acknowledgement call per receipt. Rejected as a second
round-trip with no additional guarantee.

### D7. Operator role

Decision: the backend holds the operator allowlist and returns `role` in the connect
answer. The list's source of truth is `klondiked.json`: at every start the backend
demotes every operator, then promotes the listed identities (creating their records
if unknown), so a removal takes effect at the next start and nothing else can grant
the role. The game refuses a cockpit RPC
from a non-operator before any backend call (`unauthorized`, audited through the
backend by a `security` command so the attempt is on the record), and the backend
re-checks the actor's role on every cockpit command.

Rationale: FR-011, FR-012; the role is a logical fact, so the backend owns it, and the
double check keeps a spoofed client from reaching the backend at all. The engine role
API (`PlayerRoleManagerApi`) stays for its own spike (TECHNICAL-DESIGN 19.12).

### D8. One in flight per player

Decision: `TK_BackendComponent` keeps a map from player id to the pending operation id.
A second `RpcAsk_*` command while one is pending is refused `busy` without a backend
call. The slot clears on the callback, on timeout, and on disconnect.

Rationale: clarification 3, FR-019.

### D9. Receipts and results to the client

Decision: results go only to the requesting owner (`RpcDo_OwnerCommandResult`), the
target's new balance and receipt go only to the target owner (`RpcDo_OwnerReceipt`),
and nothing about wallets is ever broadcast. No `[RplProp()]` is used anywhere in this
feature.

Rationale: visibility class "current player" for money and "privileged cockpit" for the
rest (TECHNICAL-DESIGN 15.1).

### D10. Cockpit panel

Decision: one menu (`TK_CockpitMenu`, preset `TK_CockpitMenu`), opened by the
`TK_OpenCockpit` action. Operators see boundary state, wallet lookup by player, the
credit/debit form with a mandatory reason, the two runtime values, and the last twenty
audit rows; everybody else sees their own balance only. The panel asks the server on
open and on refresh; nothing is pushed to it unrequested except the owner's receipts.

Rationale: clarification 1, FR-018, RULES 17.3; one menu instead of two keeps the layout,
the input context and the key binding single.

Key: `keyboard:KC_U`, chosen by the operator on 2026-09-03. Klondike is never loaded
together with the lobby addon, so its U binding is no conflict.

### D11. Configuration split

Decision: `$profile:TK_Backend.json` holds the base URL (bootstrap) and the development
identity flag (FR-020). Everything the backend owns (port to listen on, operators,
answer-time limit, re-check interval) lives in the backend: a small `klondiked.json`
next to the binary is read at every start; its operator list is applied each time
(D7), its two runtime values only when the database is new, after which the cockpit
`config.set` command changes them with a new configuration revision and an audit
entry. A seed value outside the `config.set` range refuses to start.

Rationale: TECHNICAL-DESIGN 3 and 3.2 (versioned configuration with recorded changes),
constitution testability (runtime regulation without restart), and the `$profile`
allowance in 2.5.

### D12. Demo-world controls

Decision: the backend accepts `-announce-contract <string>` (announce a different
contract version) and `-delay <duration>` (delay every command answer; health and
reads stay instant, or a join burst times out the engine's serialised queue) as
runtime flags, and
readiness is produced by stopping and starting the process. Runtime values change from
the panel.

Rationale: FR-017, user story 4. These are backend flags, not game code paths, so
production carries no test branch in script.

### D13. Money representation

Decision: integer whole units; the credit/debit amount is a signed integer, positive
credits, negative debits. The shared range is the signed 32-bit range of the script
`int`: the backend stores 64-bit but refuses (`invalid_amount`) any amount or resulting
total the game could not carry in an RPC or show.

Rationale: RULES 7 never mentions fractions; integers avoid rounding disputes in a
ledger. Assumption recorded in the spec.

### D14. Time

Decision: the backend stamps every record with UTC RFC 3339 and the game displays the
string it receives.

Rationale: TECHNICAL-DESIGN 2.4.

## What is deliberately not built

- No retry loop in script. A refused or timed-out command is shown; the player or
  operator retries by hand with the same operation id kept by the panel until it
  resolves (FR-006).
- No local cache of balances beyond the last displayed value.
- No RCON, website, backup, or persistence work.
- No engine role API; the allowlist stands until spike 12.
