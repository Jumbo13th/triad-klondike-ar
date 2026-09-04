# Implementation Plan: Backend Boundary

**Branch**: `001-backend-boundary` | **Date**: 2026-09-03 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-backend-boundary/spec.md`

## Summary

Stand up the boundary every later feature crosses: a Go service on the game machine
that is the system of record for the player record, wallet, ledger, receipts, audit
and the two runtime values, and a game-mode component that checks its contract
version, keeps a readiness state, sends typed commands with client-minted operation
ids, and applies nothing until the backend has accepted. The first command is operator
wallet compensation, exercised from a minimal cockpit panel; players see their own
balance. Decisions and their sources are in [research.md](research.md).

## Technical Context

**Language/Version**: Enforce Script for Arma Reforger 1.8.0.10 (game side); Go at the
version `backend/go.mod` names (see `backend/README.md`)

**Primary Dependencies**: game: vanilla `RestApi`/`RestContext`/`RestCallback`,
`JsonApiStruct`, `SCR_BaseGameModeComponent`, owner RPCs, `MenuBase`; backend: Go
standard library, `modernc.org/sqlite`

**Storage**: SQLite file in the backend data directory (WAL, `synchronous=FULL`); the
game stores nothing but `$profile:TK_Backend.json` (base URL, development identity
flag)

**Testing**: backend: `go test` domain tests (idempotency, revision, funds, payload and
identity mismatch, money range, config and seed range); game: the demo world by hand
per [quickstart.md](quickstart.md), including game-server loss before, during and after
a command (Scenario C). Measured loopback timing and physical-stage recovery are the
durability spike of TECHNICAL-DESIGN 19, not this feature.

**Target Platform**: dedicated Arma Reforger server and the backend on one host, the
backend bound to 127.0.0.1; the demo world is a local dedicated server

**Project Type**: game addon plus a co-located service

**Performance Goals**: a balance query or a compensation answers the player within 1 s
(SC-001); one health poll per re-check interval; one HTTP call per consequential command

**Constraints**: outbound HTTP only, payload under 1 MB, no custom headers, callbacks
strong-referenced; RPCs at most 8 arguments; no `[RplProp()]`; nothing broadcast;
fail closed; every visible string localized EN and RU

**Scale/Scope**: 127 players, at most one pending command per player, audit paged 20
rows at a time; 5 game scripts, 2 layouts, 2 config overrides, 1 string table, 1 Go module

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / section | How this plan satisfies it | Status |
|---|---|---|
| I Engine fidelity | Every engine call is listed with its source file in research.md; the HTTP pattern is the shipped lobby addon's, copied as is | pass |
| II Ask, never assume | Three spec clarifications, the design revision, the storage choice (SQLite) and the cockpit key (U) were decided by the operator on 2026-09-03; nothing else is open | pass |
| III KISS | One component for transport and readiness, one player component for RPCs, one menu; no base classes, registries or retry frameworks | pass |
| IV YAGNI | Four command types, all with callers in this feature; no reserved fields; `reserved` in the wallet exists because the record shape is fixed by the design (4.1) | pass |
| V Server authority and replication | See the replication classification below; zero `[RplProp()]`, zero broadcast, every `RpcAsk_*` re-validated on the server, `_S` suffix on server-only methods | pass |
| VI Localhost backend | Loopback-only listen, game is the only caller, versioned contract checked at boot, fail closed with stable reasons, no local journal | pass |
| VII Comments and hygiene | `TK_` prefix on every type; comments only on public members stating constraints; no other code base named | pass (checked at implementation) |
| Engine and asset constraints | Files the operator must double-check are listed under Workbench steps; both conf overrides carry the base game's GUID; every new GUID is generated fresh and grepped against the addon and vanilla data before handover (result in handover.md). Two `.et` files are agent-written on the operator's instruction and stated as such: `TK_GameMode.et` (derived from `GameMode_Base.et`, adds `TK_BackendComponent` and the respawn/faction/loadout children the base needs) and `TK_PlayerController.et` (derived from `DefaultPlayerControllerMP.et`, adds `TK_PlayerComponent`); the operator double-checks both in Workbench | pass |
| Persistence and data | Two stores, one split: backend owns every logical record; the game persists nothing for this feature; mutations carry actor, operation id, target, expected revision | pass |
| Documentation and language | String table with EN and RU written from meaning; reason codes are keys; `TECHNICAL-DESIGN.md` v1.1 already revised | pass |
| Testability and the demo world | Quickstart scenarios A–F; runtime values changed from the panel; backend flags produce every boundary state; demo world is a local dedicated server | pass |
| Backend logging (VI, v1.2.0) | one line per start, connect, command (op id, type, actor, target, outcome, reason code), unreadable request; no health or read lines | pass |
| Development workflow | Rules implemented: 7.1 (money is personal, survives death, removed by wipe), 17.2 (receipts, refusal reasons), 17.3 (cockpit exists), 17.4 (cockpit commands are domain operations with audit), 17.5 (compensation is an explicit operation). Design sections: 2.1, 2.3, 2.5, 4, 4.2, 4.3 step 2, 15.1, 16.1, 16.4 | pass |

### Replication classification (Principle V)

| Value | Visibility | Mechanism | Trigger |
|---|---|---|---|
| Boundary state, contract seen, config revision, runtime values | privileged cockpit | `RpcDo_OwnerCockpitState` | on `RpcAsk_OpenCockpit` and refresh |
| Own wallet | current player | `RpcDo_OwnerWallet` | on request |
| Another player's wallet | privileged cockpit | `RpcDo_OwnerWallet` | on request, operator only |
| Command result | requesting player | `RpcDo_OwnerCommandResult` | on backend answer or game-side refusal |
| Receipt and new total | affected player | `RpcDo_OwnerReceipt` | on connect (claimed) and on acceptance |
| Audit rows | privileged cockpit | begin / row × ≤20 / end | on request |
| Player role, uuid, pending op | server only | none | connect to disconnect |

No property replication, no JIP state: everything is requested by the owner after it
authenticates, exactly the "scoped snapshot after authentication" pattern of 15.2.

### Post-design re-check

Phase 1 artifacts add no replicated value beyond the table above and no record beyond
data-model.md. The one deviation from the template is that the backend's automated
tests are the only automated tests; the game side has none, as the constitution states.

## Project Structure

### Documentation (this feature)

```text
specs/001-backend-boundary/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── backend-http.md
│   └── game-rpc.md
└── tasks.md              # /speckit-tasks output
```

### Source Code (repository root)

```text
backend/
├── go.mod
├── cmd/klondiked/main.go            # flags, listen guard, seed, serve
└── internal/
    ├── api/                          # handlers for the four routes, JSON envelopes
    ├── domain/                       # commands, reason codes, invariants (+ tests)
    └── store/                        # SQLite schema, transactions, queries

Triad Klondike/
├── addon.gproj                       # string table registration (operator double-checks)
├── Configs/System/
│   ├── chimeraMenus.conf (+ .meta with the base GUID)      # TK_CockpitMenu preset
│   └── chimeraInputCommon.conf (+ .meta with the base GUID) # TK_OpenCockpit, TK_CockpitContext
├── Language/
│   ├── tk_localization.st (+ .meta)
│   └── tk_localization.<lang>.conf × 13 (+ .meta)
├── UI/Cockpit/CockpitPanel.layout (+ .meta)
└── scripts/game/
    ├── Core/
    │   ├── TK_GameMode.c              # SCR_BaseGameMode subclass; opens the cockpit key
    │   ├── TK_BackendContract.c       # JsonApiStruct request/response shapes, reason codes
    │   ├── TK_BackendComponent.c      # readiness poll, connect, command dispatch, pending map
    │   └── TK_PlayerComponent.c       # owner RPC channel on the PlayerController
    └── UI/
        └── TK_CockpitMenu.c           # the panel

tools/
└── generate-runtime-locales.ps1      # adapted from the lobby addon's generator
```

**Structure Decision**: the addon keeps the lobby addon's proven layout (`scripts/game/`
grouped by system, overrides under `Configs/System/`, one `.st` under `Language/`),
and the backend is a single Go module under `backend/` with the three packages the
feature needs and nothing else.

### Workbench steps for the operator

The prefabs and the demo world were written by the agent on the operator's instruction
(2026-09-03); every file below is to be opened and double-checked in Workbench.

1. `Prefabs/MP/Modes/TK_GameMode.et`: derived from the vanilla base game mode, adds
   `TK_BackendComponent`, the deploy-menu spawn logic, the editor faction manager
   with US and USSR made playable, and a loadout manager with one rifleman loadout
   per faction. `Prefabs/Characters/Core/TK_PlayerController.et`: the vanilla
   multiplayer controller plus `TK_PlayerComponent`.
2. `worlds/Arland/KlondikeDemo.ent` with `KlondikeDemo_Layers/a_systems.layer`
   (map, AI world, perception manager, the game mode) and `spawns.layer` (two US
   and two USSR spawn points). `Missions/KlondikeDemo.conf` is the scenario header.
3. `addon.gproj` (string table registration), `chimeraMenus.conf`,
   `chimeraInputCommon.conf`, `CockpitPanel.layout`, `CockpitTextLine.layout`,
   `tk_localization.st`; let Workbench re-index resources.
4. Build and run the backend beside the local dedicated server (Go is installed).

## Complexity Tracking

No constitution violations to justify.
