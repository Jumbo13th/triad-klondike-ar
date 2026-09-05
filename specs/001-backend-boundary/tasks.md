# Tasks: Backend Boundary

**Input**: Design documents from `/specs/001-backend-boundary/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: backend domain tests are required by the constitution (Testability; TECHNICAL-DESIGN 20.1) and are part of every backend task that changes behaviour. The game side has no automated tests; each story ends with its quickstart scenario.

**Organization**: grouped by user story so each story is independently playable in the demo world.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependencies)
- **[Story]**: US1 readiness, US2 command and wallet, US3 compensation and audit, US4 demo controls
- Paths are repository-relative; the addon root is `Triad Klondike/`

## Phase 1: Setup

**Purpose**: the two projects exist and build empty

- [x] T001 Create `backend/go.mod` (module `github.com/Jumbo13th/triad-klondike-ar/backend`, Go 1.23, require `modernc.org/sqlite`) and `backend/cmd/klondiked/main.go` with flags `-listen`, `-data`, `-announce-contract`, `-delay`, a guard that refuses any non-loopback listen address, and a `backend/README.md` (build, test, run, flags)
- [x] T002 [P] Create the addon tree `Triad Klondike/scripts/game/Core`, `scripts/game/UI`, `Configs/System`, `Language`, `UI/Cockpit`, and add `tools/generate-runtime-locales.ps1` adapted from the lobby addon's generator to read `Triad Klondike/Language/tk_localization.st`
- [x] T003 [P] Create `Triad Klondike/Language/tk_localization.st` (+ `.meta`, fresh GUID) with the English text for every key of this feature (`TK-Cockpit_*` labels, `TK-State_*` boundary states, `TK-Reason_*` for every reason code in contracts/backend-http.md, `TK-Receipt_*`), run the generator to produce the 13 runtime confs (+ `.meta` each), and register the table in `Triad Klondike/addon.gproj`
- [x] T004 [P] Add `backend/data/` and the built `backend/klondiked` binary to `.gitignore`

---

## Phase 2: Foundational

**Purpose**: schema, envelopes, components and the panel shell every story builds on

**⚠️ CRITICAL**: no user story work starts before this phase is complete

- [x] T005 Implement `backend/internal/store/store.go` and `schema.go`: open SQLite with WAL and `synchronous=FULL`, create the tables of data-model.md, seed `config` and operator `players` rows from `klondiked.json` on first run only
- [x] T006 [P] Implement `backend/internal/domain/reason.go` (status and reason-code constants) and `domain/envelope.go` (command envelope, answer, canonical payload hash)
- [x] T007 Implement `backend/internal/api/server.go`: routes for the four endpoints, JSON reply helpers, the `-delay` middleware, and `GET /v1/health` returning contract "1" (or the `-announce-contract` override), the newest config revision and its two values; `api/health_test.go` covers contract, override and config values (depends on T005, T006)
- [x] T008 [P] Implement `Triad Klondike/scripts/game/Core/TK_BackendContract.c`: `JsonApiStruct` classes for health, connect request and response, wallet, command envelope, answer, receipt and audit page; `CONTRACT_VERSION = "1"`; reason-code string constants
- [x] T009 [P] Implement `Triad Klondike/scripts/game/Core/TK_GameMode.c`: `TK_GameModeClass` and `TK_GameMode : SCR_BaseGameMode`; on non-dedicated sessions add the `TK_OpenCockpit` action listener that opens the cockpit menu
- [x] T010 Implement `Triad Klondike/scripts/game/Core/TK_BackendComponent.c` skeleton: `SCR_BaseGameModeComponent` subclass, `$profile:TK_Backend.json` load (BaseUrl, default written when missing), `RestContext` creation, a request-holder class that strong-references its `RestCallback`, boundary state enum and fields, per-player `{uuid, role}` and pending-operation maps, `GetInstance()` (depends on T008)
- [x] T011 [P] Implement `Triad Klondike/scripts/game/Core/TK_PlayerComponent.c` skeleton: `ScriptComponent` on the PlayerController, `GetByPlayerId`, `GetLocalInstance`, every `RpcAsk_*` and `RpcDo_Owner*` of contracts/game-rpc.md declared with their exact signatures and dispatch to `TK_BackendComponent`
- [x] T012 Author `Triad Klondike/Configs/System/chimeraMenus.conf` (+ `.meta` carrying the base game's GUID `C747AFB6B750CE9A`) with the `TK_CockpitMenu` preset, and `chimeraInputCommon.conf` (+ `.meta` carrying the base GUID read from `Arma-Reforger-Script-Diff/GameData/Configs/System/chimeraInputCommon.conf.meta`) with action `TK_OpenCockpit` on `keyboard:KC_U` and context `TK_CockpitContext`; every new object id fresh
- [x] T013 [P] Author `Triad Klondike/UI/Cockpit/CockpitPanel.layout` (+ `.meta`, fresh GUID) with named widgets `StateText`, `ContractText`, `ConfigText`, `PlayerBalanceText`, `OperatorSection`, `WalletUuidEdit`, `WalletResultText`, `AmountEdit`, `ReasonEdit`, `CreditButton`, `RetryButton`, `ResultText`, `TimeoutEdit`, `IntervalEdit`, `ApplyConfigButton`, `AuditList`, `AuditMoreButton`, `RefreshButton`, `CloseButton`; all texts as `#TK-` keys; buttons on `SCR_ButtonBaseComponent`
- [x] T014 Implement `Triad Klondike/scripts/game/UI/TK_CockpitMenu.c` shell: `modded enum ChimeraMenuPreset { TK_CockpitMenu }`, `MenuBase` subclass, `Open()`, widget lookup and button wiring, `OperatorSection` hidden unless the role answer says operator, Escape closes (depends on T011, T013)

**Checkpoint**: both projects build; the panel opens on U and shows placeholders

---

## Phase 3: User Story 1 - Readiness (Priority: P1) 🎯 MVP

**Goal**: the server proves the backend and its contract at boot and keeps proving it; operators see the state and reason

**Independent Test**: quickstart Scenario A

- [x] T015 [US1] Implement the readiness poll in `Triad Klondike/scripts/game/Core/TK_BackendComponent.c`: first `GET /v1/health` on server start, repeat with `CallLater` at the current interval, parse with `TK_BackendContract`, set `READY` / `UNREACHABLE` / `VERSION_UNKNOWN`, apply `SetTimeout` and reschedule when the answer changes the two values, log every state transition once
- [x] T016 [US1] Implement `IsReady_S()` and `NotReadyReason_S()` in `TK_BackendComponent.c` for every later command path
- [x] T017 [US1] Implement `RpcAsk_OpenCockpit` → `RpcDo_OwnerCockpitState` in `TK_PlayerComponent.c` and the state, contract and config display in `TK_CockpitMenu.c` (`#TK-State_*` keys, refresh button re-asks)
- [x] T018 [US1] Play quickstart Scenario A on the local dedicated server and record the outcome in `specs/001-backend-boundary/quickstart.md` under the scenario

**Checkpoint**: readiness is visible and recovers without a restart

---

## Phase 4: User Story 2 - One definitive answer (Priority: P1)

**Goal**: identity on audit success, player record and wallet, the command envelope with idempotency, and a player's own balance

**Independent Test**: quickstart Scenarios B steps 1–2 (query side), C, D

- [x] T019 [US2] Implement `POST /v1/players/{uuid}/connect` in `backend/internal/store` and `api`: one transaction that creates player and zero wallet when absent, updates name and last connect, returns role, wallet and undelivered receipts and marks them delivered; `api/connect_test.go` covers idempotent create and receipts claimed exactly once
- [x] T020 [P] [US2] Implement `GET /v1/players/{uuid}/wallet` with `player_unknown`; `api/wallet_test.go`
- [x] T021 [US2] Implement `backend/internal/domain/commands.go`: operation lookup by `op_id` (same hash → recorded answer as `already_applied`; different hash → `op_id_payload_mismatch` plus a `security` audit row), type dispatch, one audit row for every outcome including refusals, refused operations recorded so a repeated refusal replays; `domain/commands_test.go` covers all three outcomes
- [x] T022 [US2] Implement `OnPlayerAuditSuccess` in `TK_BackendComponent.c`: read the identity through `GetGame().GetBackendApi().GetPlayerIdentityId`, treat empty as not ready (log once), call connect, cache `{uuid, role}`, push each claimed receipt with `RpcDo_OwnerReceipt`; `OnPlayerDisconnected` clears the cache and the pending slot
- [x] T023 [US2] Implement `SendCommand_S` in `TK_BackendComponent.c`: identity, role (when required), busy and readiness checks in that order, `POST /v1/commands`, timeout and non-200 → `backend_unreachable`, answer parsing, pending slot cleared on every path, result delivered through the caller's owner RPC
- [x] T024 [US2] Implement `RpcAsk_GetWallet` → `RpcDo_OwnerWallet` in `TK_PlayerComponent.c` (self for anyone, another uuid for operators) and the player balance display in `TK_CockpitMenu.c`; a received `RpcDo_OwnerReceipt` updates the balance from `totalAfter`
- [x] T025 [US2] Implement operation-id handling in `TK_CockpitMenu.c`: mint with `UUID.GenV4()` on press, keep it while the result is `backend_unreachable`, `RetryButton` resends the same id, clear on any other terminal answer
- [x] T026 [US2] Play quickstart Scenarios C and D and record the outcomes (C steps 1–3 and 5, D steps 1–3 passed 2026-09-04; C step 4 covered by backend test)

**Checkpoint**: a player sees their balance; a retried command never applies twice

---

## Phase 5: User Story 3 - Operator compensation on the record (Priority: P2)

**Goal**: credit and debit with a reason, refused for the unauthorized and the overdrawn, every outcome audited and readable in game

**Independent Test**: quickstart Scenario B

- [x] T027 [US3] Implement `wallet.compensate` in `backend/internal/domain/commands.go` and the store: operator role, non-empty reason, non-zero amount, target exists, expected revision, funds; wallet, ledger, receipt and audit in one transaction; `domain/compensate_test.go` covers accept, each refusal, and the invariant `total == sum(ledger)` after a drill of 100 randomized commands with repeated operation ids (SC-003)
- [x] T028 [P] [US3] Implement the `security` command type and `GET /v1/audit` (newest first, `before` paging, limit capped at 50) in `backend/internal/domain` and `api`; `api/audit_test.go`
- [x] T029 [US3] Implement `RpcAsk_Compensate` in `TK_PlayerComponent.c` and `TK_BackendComponent.c`: non-operator → `unauthorized` to the caller and a `security` command to the backend; otherwise the envelope; on `accepted` push `RpcDo_OwnerReceipt` to the target when online
- [x] T030 [US3] Implement `RpcAsk_GetAudit` → begin / row / end in `TK_PlayerComponent.c` and the audit list with `AuditMoreButton` paging in `TK_CockpitMenu.c`
- [x] T031 [US3] Implement the operator section in `TK_CockpitMenu.c`: wallet lookup by uuid, amount and reason form (empty reason refused client-side too), result line showing the reason key and the current values from the answer
- [x] T032 [US3] Play quickstart Scenario B and record the outcome (steps 1–4 passed 2026-09-04; step 5 covered by backend test + server gate, no client path)

**Checkpoint**: the first cockpit command works end to end with audit

---

## Phase 6: User Story 4 - Driving the boundary in the demo world (Priority: P3)

**Goal**: every boundary state and both runtime values can be produced without files or restarts

**Independent Test**: quickstart Scenarios A steps 3–5, C step 1, E

- [x] T033 [US4] Implement `config.set` in `backend/internal/domain/commands.go`: range validation (`answer_timeout_s` 1..120, `recheck_interval_s` 1..600), new `config` row, audit; `domain/config_test.go`; verify `-delay` and `-announce-contract` behave per contracts/backend-http.md in `api/flags_test.go`
- [x] T034 [US4] Implement `RpcAsk_SetBoundaryConfig` in `TK_PlayerComponent.c` and `TK_BackendComponent.c`, and the config form in `TK_CockpitMenu.c`; the next health poll applies the new values
- [x] T035 [US4] Play quickstart Scenario E and record the outcome (passed 2026-09-04, revisions 2 and 3)

**Checkpoint**: all four stories playable

---

## Phase 7: Polish

- [x] T036 Write the Russian text for every key in `Triad Klondike/Language/tk_localization.st` from the meaning (community loan words kept), regenerate the runtime confs, play quickstart Scenario F
- [x] T037 [P] GUID audit: grep every new GUID and object id of the addon against `Triad Klondike/` and `Arma-Reforger-Script-Diff/GameData/`; record the result in the handover
- [x] T038 [P] Hygiene pass over `Triad Klondike/scripts/` and `backend/`: `TK_` prefix on every type, `_S` on server-only methods, comments only on public members and only for constraints, no other code base named, no dead code
- [x] T039 Handover: the Workbench steps from plan.md, the list of asset files to double-check, the quickstart results, in the constitution's `Changes:` format → `specs/001-backend-boundary/handover.md` (2026-09-05)
- [x] T041 Development identity (clarification 2026-09-04, FR-020): `DevIdentityFromName` in `TK_BackendConfig`, `TK_DevIdentity.FromName` with the engine's non-dedicated derivation, substitution in `OnPlayerAuditSuccess` with warnings at start and per player; quickstart Scenario D step 3 and prerequisites
- [x] T042 Play quickstart Scenario D step 3 on the local dedicated server and record the outcome
- [x] T043 Repeat failed connects when the boundary turns ready (`ConnectMissing_S` from `SetState_S`), so a player who joined during a backend outage needs no rejoin; quickstart Scenario D2
- [x] T044 Play quickstart Scenario D2 and record the outcome (passed 2026-09-04)
- [x] T045 Receipt delivery (clarification 2026-09-04): `target_online` on the compensate envelope, backend marks the receipt delivered at creation when true (`TestLivePushedReceiptIsNotReplayedOnConnect`), game sets it from the target session; contract updated
- [x] T046 Verify on the local server: credit an online player, reconnect them, connect claims no receipt; credit them while away, reconnect, connect claims exactly that one (passed 2026-09-04: receipts=0 after the live credit, receipts=1 after the away credit by identity)
- [x] T040 Demo world and prefabs, written by the agent on the operator's instruction: `Prefabs/MP/Modes/TK_GameMode.et`, `Prefabs/Characters/Core/TK_PlayerController.et`, `worlds/Arland/KlondikeDemo.ent` with `a_systems.layer` and `spawns.layer`, `Missions/KlondikeDemo.conf`, the `IngameContext` binding of `TK_OpenCockpit`, three mission strings EN+RU
- [x] T047 Review of PR #1 (2026-09-05, Fable and CodeRabbit; false positives skipped, see handover.md): connect answer dropped for a player who left meanwhile; ambiguous name refused `target_ambiguous` (new string EN+RU); empty compensate target refused; receipt pushed only when the target was online at dispatch; health timer re-armed on an interval change; audit reason codes translated; operation id bound to type, actor, subject and target; money held to the 32-bit range; seed values range-checked at start; SQLite pragmas in the DSN; CI token read-only; spawn points derived from `SpawnPoint_Base.et`; string-table author fields, `backend.ps1`, and the stale spec text corrected. Played on the local dedicated server the same day, all passed (quickstart.md)

---

## Dependencies

- Phase 1 → Phase 2 → US1 → US2 → US3 → US4 → Polish. US1 needs only the health route and the poll; US2 needs connect, wallet and the envelope; US3 needs US2's envelope; US4 needs US3's cockpit section.
- Inside Phase 2: T005 → T007; T008 → T010; T011 and T013 → T014.
- Backend tasks and game tasks of the same story can proceed in parallel once their contract is fixed (contracts/ are the fixed point).

## Parallel execution examples

- Phase 1: T002, T003, T004 together after T001.
- Phase 2: T006, T008, T009, T011, T013 together; then T005/T007 and T010/T012/T014.
- US2: T019 and T020 (backend) beside T022 (game); T021 beside T023.
- US3: T027 and T028 beside T029 and T030.

## Implementation strategy

MVP is Phase 1 + Phase 2 + US1: a server that proves its backend and shows it. Each
following story adds one closed loop (query, compensation, regulation) and ends with
its quickstart scenario played on the local dedicated server before the next starts.
