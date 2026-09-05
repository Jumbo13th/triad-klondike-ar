<!--
Sync Impact Report
- Version: 1.1.0 → 1.2.0 (MINOR: guidance added, no principle redefined)
- Added: Principle VI bullet "The backend logs what it decides, not what it sees"
  (one line per decision: start, connect, command, configuration change, unreadable
  request; no health polls or reads, nothing twice).
- Previous (1.0.0 → 1.1.0): Principle I source 3 (the Conflict game mode as the
  in-game reference for prefabs, layouts and UI flows; Lite Lobby moved to source 4);
  section "User Interface"; Development Workflow bullet "Audits and reviews have done
  criteria".
- Templates requiring updates: none; plan-template.md's constitution check reads this
  file at runtime. Backend features add a "logging" row to their constitution check.
- Follow-up TODOs: none. Feature 001's backend was brought in line with the logging
  bullet on 2026-09-04.
-->

# Triad: Klondike Constitution

Triad: Klondike is a persistent resource-war game mode for Arma Reforger, written in
Enforce Script under the `TK_` prefix. `docs/RULES.md` owns player-observable behaviour,
`docs/TECHNICAL-DESIGN.md` owns the implementation contract, and this constitution owns
how the code is written. Where this document and the technical design overlap, the
technical design is more specific and this document is more binding.

## Core Principles

### I. Engine Fidelity — Never Invent Arma Code

Every engine call, component, attribute, event, and replication construct MUST be
traceable to one of these sources, in this order of authority:

1. The official Arma Reforger scripts and game data for the pinned game version
   (`Arma-Reforger-Script-Diff/`, tagged per release — check out the shipped tag).
2. The official Enfusion documentation for replication, persistence, components,
   serialisation, and configuration.
3. The Conflict game mode inside those official sources (`scripts/Game/Campaign/`,
   `GameData/Prefabs/MP/Modes/Conflict/`, `GameData/UI/layouts/Campaign/`,
   `HUD/CampaignMP/`, `Tasks/Conflict*`, the deploy menu and the map) for how a
   persistent multiplayer mode is assembled in this engine: its prefabs, layouts,
   menus, HUD elements and player flows are the closest thing to Klondike that ships
   with the game, and are the first place to look for a prefab, a layout or a UI flow.
4. Lite Lobby (`lite-lobby-ar/`), the community's shipped addon, for patterns already
   proven on this community's servers at 127 players.

Rules:

- If a pattern does not exist in those sources, it is not known to work. Do not
  extrapolate an API from its name, from another engine, or from how it "should" behave.
- Before implementing any feature, search the official scripts for the vanilla
  equivalent (a manager, a user action, a serializer, a UI flow) and follow its shape.
- Working reference code is copied as it is. Do not "fix", restyle, or improve a pattern
  that already runs in production; its quirks may be what makes it work.
- Never browse the game installation, the workshop cache, or the player profile
  directory. The in-repository sources above are the only sanctioned references.

Rationale: a previous rewrite in this community failed on assumptions about how the
replication layer behaves. Every assumption made things worse. The engine is the
ground truth, and the game scripts are the only complete documentation of it.

### II. Ask, Never Assume

When a technical question cannot be answered from the sources in Principle I, STOP and
ask the operator. Do not guess, do not pick silently, do not reinvent.

- Browse existing code first, then present the options found with arguments for each,
  and let the operator decide.
- This applies to every technical decision, not only replication: engine APIs,
  Workbench behaviour, ambiguous rules text, and anything that looks wrong in code that
  currently works.
- A thirty-second question is always cheaper than a debugging session.

### III. KISS — Keep It Simple

Prefer one clear state machine and one safe server command over several overlapping
frameworks. The launch target is deliberately small.

- Choose the simplest design that satisfies the rule being implemented. Complexity MUST
  be justified in the plan by a rule in `RULES.md` or an invariant in
  `TECHNICAL-DESIGN.md`, never by "flexibility".
- No abstraction layers, base classes, registries, or plugin systems with a single
  concrete use.
- Fewer features done well beat many features done poorly. Dropping a feature is a valid
  answer, and the correct one, whenever implementing it would harm network performance
  or durability at the target scale. Present the trade-off; the operator decides.

### IV. YAGNI — Build Only What Is Required

Nothing is implemented unless it is required by `RULES.md`, by an invariant in
`TECHNICAL-DESIGN.md`, or by an accepted feature specification.

- No speculative configuration knobs, "reserved for future" fields, optional code paths,
  or generalisations for scenarios nobody has specified.
- No helper, utility, or convenience API that has no caller in the current task.
- An agent that sees a "useful" addition outside the task MUST mention it and MUST NOT
  build it.
- A mechanism not required by the rules is out of scope, exactly as the technical design
  states.

### V. Server Authority and Replication Discipline

The dedicated server is authoritative for every gameplay decision. Replication is the
scarcest resource in this project and every byte on the wire is a design decision.

- Clients request; the server validates every argument and permission and decides. The
  client never supplies a trusted price, quantity, faction, yield, capacity, rank, timer
  expiry, or entitlement. Server-only methods carry the `_S` suffix.
- Follow the vanilla replication model: `[RplProp()]` for scalars only, RPCs
  (`RpcAsk_*` client→server, `RpcDo_*` server→clients) for deltas, `RplSave`/`RplLoad`
  for join-in-progress. Never `[RplProp()]` on a collection.
- `Replication.BumpMe()` is called on the authority only, after changing a replicated
  scalar, and never on a component that carries large replicated state.
- RPC methods take at most 8 arguments; split beyond that, and do not send fields that
  are guaranteed defaults at creation time.
- Per-entity replication multiplies by the entity count. Centralise state in managers;
  world entities carry zero replication beyond what vanilla provides unless a plan
  justifies it.
- Classify every payload by visibility before implementing it (public, current player,
  faction-scoped, Local-only, vehicle owner, leaseholder, privileged cockpit). Never send
  a secret to every client and rely on the UI to hide it.
- Large maps of wallets, bids, items, vehicles, or audit rows are never one continuously
  changing property. Use small public summaries, scoped snapshots after authentication,
  revisioned deltas after committed mutations, and pagination.
- `RplSave` and `RplLoad` MUST mirror each other exactly; a mismatch corrupts silently.
  Arrival order of replicated items is not deterministic; sort by a server-assigned
  index, never by arrival.
- Do not derive a remote player's state from local entity inspection inside an RPC
  handler; it races entity replication. Anything keyed by player id is initialised from
  `OnPlayerConnected`, not from component init.
- Instrument first, then change, then measure at the target player count. One
  replication change per test.

### VI. Localhost Backend — One Machine, One Boundary

A Go service running on the same machine as the dedicated game server is the game's
backend and the system of record for every logical record: wallets and ledgers, bids and
escrow, leases, vehicle and plot registrations, weekly results, receipts, and audit. The
game server owns live gameplay and the physical world; it does not own money or
contracts. This boundary runs through every feature and every plan.

- All traffic between the game and the backend goes over the loopback interface. The
  backend never listens on a public interface, and the game never calls any other host
  during play. Latency, reachability, and exposure are engineered out by co-location,
  not by retries.
- The engine's REST client is outbound-only, with a small payload limit, and there is no
  inbound listener in script. The game calls the backend; the backend never pushes.
  Anything the backend must tell the game is returned in a response or polled.
- Every consequential transaction is one backend call carrying the typed command of
  Principle V: operation id, actor and subject, target stable id, expected revision, and
  only the choice the server cannot derive. The backend answers accepted, refused with a
  stable reason, or already applied. The game applies the physical effect only after the
  backend accepted, and never before.
- Fail closed. When the backend does not answer, consequential transactions are refused
  with a stable reason the player can see; ordinary play (movement, combat, cutting,
  item movement) continues. The game never journals money locally to replay later.
- The website synchronises with the backend, never with the game. The game server knows
  nothing about the website.
- The backend contract is versioned. At boot the game checks the version and refuses
  consequential play against a contract it does not know.
- The backend logs what it decides, not what it sees. One line per decision: every
  start with the values it runs on, every connect with identity and role, every
  command with operation id, type, actor, target, outcome and reason code, every
  configuration change, and every request it could not understand. Health polls and
  reads are not logged, no event is logged twice, and a line carries only what an
  operator scanning a console needs to know what happened and why; the audit table
  holds the detail. Logging that does not answer "what did the backend do and why" is
  noise and is not written.

### VII. Comments and Code Hygiene

Code explains itself; comments exist only for what code cannot show.

- Comments are written at the level of public members: a class, a public method, an
  editor attribute, or a replicated property. Not on private helpers, not inside method
  bodies, not above self-explanatory lines.
- A comment states a constraint or a WHY that the reader cannot recover from the code:
  an ordering requirement, an engine quirk, a replication cost, a rule reference. One or
  two lines. Walls of text, narration of the next statement, editorial judgements,
  restated documentation, and progress notes are forbidden.
- Comments and identifiers MUST NOT name any code base other than the official Arma
  Reforger scripts. Other addons and projects may be read as references (Principle I),
  but their names, class prefixes, and provenance never appear in shipped code. Write
  the actual reason instead of "as done in X".
- Every type, method, and prefab authored by this project carries the `TK_` prefix.
  Modded vanilla classes keep vanilla parameter names in overrides exactly.
- No dead code, no commented-out code, no `_OLD` folders inside the addon.

## Engine and Asset Constraints

Scripts own behaviour. Workbench owns audio-graph node ids and is the final check on
every non-script asset an agent writes.

- `.layout`, `.meta`, `.conf`, and `.acp` files may be written by an agent. Every such
  file is listed in the handover for the operator to open and double-check in Workbench.
- `.et` prefabs are the one asset class to be sceptical about. An agent may touch them
  only when the change is small and fully understood (adding a known component, setting
  a known attribute), MUST say so explicitly, and MUST ask the operator to double-check
  the file in Workbench before anything else builds on it. New prefabs are derived from
  existing vanilla assets (a rock prop, a hand tool, a carryable item), never modelled or
  animated from scratch.
- The engine has an override mechanism: Workbench's "Create Override" on a base-game
  resource produces a file at the same path carrying the base game's GUID, and the engine
  merges it into the original. This is the sanctioned way to add components to vanilla
  prefabs (for example the character base) and entries to system configs
  (`chimeraMenus.conf`, `chimeraInputCommon.conf`). A standalone file with a fresh GUID at
  the same path is a separate resource the engine never merges; it builds fine and
  silently does nothing.
- Never override a global vanilla resource that changes behaviour for every mod on the
  server (for example the vanilla VoN audio graph). A feature whose only implementation
  path is such an override is not implementable here; propose dropping it.
- Never hand-author nodes in `.acp` audio graphs; hand-written nodes render but are dead
  at runtime. Attribute and wire edits on Workbench-born nodes, and verbatim copies of
  Workbench-born blocks, are allowed.
- Agents generate `.meta` files and their resource GUIDs. A GUID is 16 random
  uppercase hex digits, generated fresh for each resource, never derived from a name, a
  counter, or a neighbouring file, and never copied from another addon or from the
  vanilla game. Before handover, grep the whole addon and the vanilla game data for every
  new GUID to prove it does not intersect anything. The only GUID that is deliberately
  reused is the base game's own, in an override file (see above).
- Scripts and configs reference a resource by the GUID in its `.meta` file. If Workbench
  reassigns a GUID on import, re-read the `.meta` and re-sync every reference to the
  actual value rather than fighting the tool.
- Every object id in hand-authored `.layout` and `.conf` content follows the same rule:
  freshly generated and unique. Ids are copied only when an include must address an
  existing vanilla object.
- Modded `BaseContainerProps` script objects repeat the decorator and base class, or the
  prefab's component list silently truncates.

## User Interface

Every screen, panel, HUD element and notification this project ships is used by
players and operators who already know Arma Reforger. It must look and behave like
the game they know, and it must be pleasant to use. A bare technical panel is not an
acceptable operator interface.

- Follow the visual language of Arma Reforger: the fonts, colours, spacing, button and
  edit-box behaviour, focus and hover states, list and scroll conventions, and the
  layout grammar of the vanilla menus. Build screens from the vanilla widget library
  (`GameData/UI/layouts/WidgetLibrary/`, `WidgetLibraryExtended/`, `Common/`) and the
  `SCR_*` UI components rather than raw widgets with hand-picked colours.
- The Conflict game mode is the reference for how a screen of this kind is done in the
  game (Principle I, source 3): the deploy menu, the map and its side panels, the task
  list, the base and service panels, the HUD. Before designing a screen, open the
  Conflict layout that does the closest job and follow its structure; Lite Lobby's
  screens are the second reference.
- Design from the user's task, not from the data model. An operator screen groups what
  the operator does together (look up, act, confirm, see the result) with clear labels,
  sensible defaults, visible state, readable feedback, and no free-text where a choice
  or a list will do. A player screen shows what the player needs at the point of use
  and nothing else.
- Do not repeat this project's own earlier screens or the agent's own habits as if they
  were a style. Each screen is judged against the game's screens, not against the
  previous agent-made one.
- A screen is reviewed in the running game, at the game's resolution and scale, before
  it is called done: alignment, clipping, truncated strings in both languages, keyboard
  and controller focus, and Escape behaviour.

## Persistence and Data

- Two stores, one fixed split. The localhost backend (Principle VI) is the system of
  record for logical records. The native persistence system stores the physical world
  only: entities, inventories, vehicles, structures, and the stable ids that link them
  to backend records. No fact is owned by both stores; the linking id is the only thing
  they share.
- Every mutation enters through a typed domain command carrying actor, operation id,
  target stable id, and expected revision. Retrying an accepted operation returns its
  recorded result; stale revisions refresh and retry, never best-effort write.
- Consequential transactions (money, item conversion, registered property, settlement,
  operator compensation, wipe) are backend calls, produce a receipt keyed by operation
  id, and change the physical world only after acceptance. Ordinary item movement uses
  the native periodic checkpoints and creates no backend call.
- Boot reconciles the restored physical world against the backend by operation id. A
  mismatch is resolved by a documented compensation rule, logged, and visible in the
  cockpit; it is never guessed and never silently repeated.
- Backend identity is the stable server identity, never a transient player id or
  replication id.
- No numeric checkpoint budget, round-trip budget, or throughput figure is written into
  design or code before the persistence spike measures it on the packaged mission with
  the backend running alongside. An unflushed journal is never presented as a durable
  fallback.
- Nothing beyond the backend is reachable from the game during play. Any website or
  export is downstream of the backend and read-only with respect to the game.
- Mutable balance values live in validated server configuration and are displayed in
  game; they are never hard-coded and never copied into player documents.

## Documentation and Language

- Source order when documents disagree: `RULES.md`, then `TECHNICAL-DESIGN.md`, then the
  validated server configuration, then the live in-game interface, then the generated
  PDFs. An implementation shortcut may not silently change a rule; a design change
  updates the sources in that order.
- Every normative rule in `RULES.md` is numbered by section; code and specs cite rules
  by that number.
- The project is localized. English is the source language of code and documents;
  Russian is the language of the community and the first target, and other languages
  may follow. Everything shown on screen (UI, notifications, receipts, refusal reasons,
  admin cockpit, briefings, lore) ships with a Russian translation. No user-visible
  string is hard-coded; every string goes through the string table so it can be
  translated.
- Russian is written as native Russian, the way players actually speak and write, not
  as a rendering of the English sentence. Machine translation and literal translation
  are forbidden. Before a Russian string is accepted, read it as a player would: if it
  sounds like a translated manual, rewrite it from the meaning. Community loan words
  stay as players use them (вайп, респавн, лут, кулдаун); descriptive Russian
  substitutes for them are not invented.
- Russian and English editions of a document are authored from the same decisions,
  not from each other.
- The atmosphere of the setting is shown, never labelled. Internal shorthand for the
  tone does not appear in lore, UI, or briefings. Rejected design directions recorded in
  the rules appendix are not reopened without the operator.
- Documents record the why and the dead ends; git history records the what.

## Open Source Standards

This repository is public. Everything committed is read by strangers who never saw the
conversation that produced it, and it represents the community.

- Follow ordinary open-source conventions: a README that says what the project is and
  how to build it, a licence file, a conventional layout, small focused commits with
  plain messages, and a history that reads without the chat log.
- Nothing private leaks: no credentials, tokens, server addresses, player data,
  machine-local paths, or reference material the project does not own (Principle VII).
- No AI slop. Every file, paragraph, and comment earns its place by telling a reader
  something they need. Forbidden: boilerplate and filler, overview documents that
  restate other documents, placeholder sections, hedging, inflated adjectives such as
  "comprehensive" or "robust", emoji, decorative headings, badges, changelogs generated
  from diffs, and text written to look thorough rather than to be read.
- One source per fact. A rule, a term, or a value lives in exactly one document; other
  documents link to it rather than repeat it.
- A document that cannot say who needs it and when is deleted, not kept "for context".
- The repository stays clean. No legacy artifacts, no `_OLD` folders, no superseded
  documents, no generated review files, no scratch notes, no dead build scripts, no
  drafts kept next to the thing they became. When something is replaced, its
  predecessor is deleted in the same change; git history is the archive.

## Testability and the Demo World

There is no automated way to test game functionality. The only place a feature can be
exercised is a demo world played by hand, and the demo world is the acceptance
environment for everything.

- Every feature MUST be reproducible in the demo world. A feature that can only be
  observed on a live season server with real players, real time, or real economy
  history is not testable and is not done. The plan for each feature names the demo
  world setup it needs (placed entities, configured values, seeded state).
- Every value that shapes a feature's behaviour (prices, yields, timers, caps, radii,
  cooldowns, income rates, week and Vakhta lengths) MUST be regulated by the admin at
  runtime from the admin dashboard, without a restart or a config redeploy, so that a
  tester can drive the feature through its states in minutes instead of days.
- Every feature exposes admin-driven controls for its state: force a transition, seed or
  reset a record, grant or remove money and items, trigger a scheduled job now, expire a
  timer, spoof a count. These controls are part of the feature, planned and built with
  it, gated to the privileged cockpit role, and audited like any other operator command.
  They are not debug leftovers to remove before launch; they are how the game is tested.
- Time-driven mechanics are designed against server time supplied through the same
  admin controls, so a week rollover, a lease expiry, or an auction close can be made to
  happen on demand.
- Admin controls follow the same authority and replication rules as player commands
  (Principle V). They are never a client-side shortcut around the server.

## Development Workflow

- The rules and the technical design were written before implementation started, and
  they are general and wide by nature. Many things will change once code meets the
  engine. Every specification therefore starts by evaluating the current state of the
  solution: what already exists, what the engine has since proven or disproven, and
  whether the rule it implements still makes sense as written. Where it does not, the
  spec revises the game mechanic and its technical implementation and proposes the
  corresponding `RULES.md` and `TECHNICAL-DESIGN.md` changes, with arguments, for the
  operator to decide. Rules are not reinterpreted silently, and they are not implemented
  literally when the literal reading is wrong.
- The project is Spec Kit driven, and the way of working is fully aligned with it.
  Every feature moves through the Spec Kit phases in order: specify, clarify, plan,
  tasks, implement, with analyze and checklist wherever the templates call for them. No
  code is written for a feature without an accepted specification and plan, and no task
  is implemented that is not in the task list.
- The principles of Spec Kit apply in full. The specification states what and why with
  no implementation detail; the plan states how and carries the constitution check;
  tasks are small, ordered, and individually verifiable; implementation follows the
  task list rather than the conversation. The artifacts under `specs/` are the contract
  for a feature, so a change of mind updates the artifact before it changes the code.
  Skipping a phase, merging phases, or editing artifacts out of order is a defect, not a
  shortcut. The `.specify/` templates, scripts, and memory are the tooling for this
  flow; they are kept in sync with this constitution and never bypassed with ad-hoc
  documents.
- Every plan carries a constitution check that names the rule numbers it implements,
  classifies every new replicated value by visibility and mechanism (Principle V), and
  lists the Workbench steps the operator must perform.
- Ambiguities discovered mid-task are collected as questions for the operator, not
  resolved by assumption (Principle II).
- Audits and reviews have done criteria. Before an audit, analysis, review or
  checklist pass starts, it states what "done" means: the principles and rule numbers
  it checks, the artifacts it covers, and the finding types it reports. It runs once
  over that scope, reports, and stops. Searching for issues without a stated finish
  line is forbidden: there is no "one more pass", no widening of scope mid-audit, and
  no re-audit of what already passed unless the operator names a new criterion. Fewer
  criteria checked to the end beat many checked forever.
- Each change is played through in the demo world, using the admin controls it ships
  with, before it is considered done; scale-sensitive changes are additionally measured
  at the target player, item, vehicle, and storage counts. Telemetry (RPC rate,
  checkpoint duration, queue length) is added before a change is tuned, not after.
- Launch is gated by the spikes named in the technical design. A failed spike blocks
  launch; it does not license a workaround.
- The operator imports assets, places entities, and publishes. Agents hand over a
  precise list of those steps, and of every asset file to double-check, with every
  feature.
- Never commit, push, tag, or otherwise change repository history without the operator's
  explicit permission for that specific change. Staging is fine; committing is not.
- Every summary of work, whether a handover message, a commit message, or a pull request
  description, uses this format and nothing more:

  ```text
  Changes:
  - Short change A
  - Short change B
  ```

  One line per change, stating what changed. No preamble, no rationale essay, no
  headings, no "why this matters" paragraphs. The operator does not read complex pull
  request descriptions or walls of text from agents; a description that needs more than
  the list is a sign the change should be split.

## Governance

This constitution supersedes every other practice in the repository. Feature
specifications, plans, and task lists MUST be checked against it, and a plan that
violates a principle is rejected or amended before implementation starts.

Amendment procedure: propose the change with its rationale, update this file, record
the change in the Sync Impact Report at the top, and bump the version:

- MAJOR — a principle removed or redefined in a backward-incompatible way.
- MINOR — a principle or section added, or guidance materially expanded.
- PATCH — clarifications and wording that do not change meaning.

The operator ratifies every amendment. Rules in `RULES.md` and constraints in
`TECHNICAL-DESIGN.md` change through their own source order; this document only
changes when the way of working changes.

**Version**: 1.2.0 | **Ratified**: 2026-09-03 | **Last Amended**: 2026-09-04
