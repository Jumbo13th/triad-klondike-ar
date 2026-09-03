# Triad: Klondike — Mode Rules (v1.0)

> This is the canonical English source for player-observable rules. The English and
> Russian PDFs are review editions of the same design. The technical implementation is
> specified separately in `TECHNICAL-DESIGN.md`.
>
> Rules are numbered by section (`3.4` is the fourth rule of section 3). The numbers are
> stable references for review and for the PDF editions, not an order of importance.
>
> Prices, durations, yields, capacities, limits, fees, schedules, and other balance
> values belong to server configuration. The game shows the current values where a
> player needs them. This document defines how the rules behave rather than freezing
> playtest numbers into prose.

## 1. The Island and the Game

Triad: Klondike is a persistent, full-loot resource war on Kolguyev. Meridian Group and
Granit Concern fight for the island's mines. The Locals live by entering ground they
cannot own, and the whitelisted Garrison is an armed wildcard outside the economic
competition.

**1.1 The player journey.** The loop is deliberately physical:

1. Obtain a cutter, equipment, and transport.
2. Gain access to a mine, or enter somebody else's mine as a Local.
3. Cut kolguyevite by hand at a working face.
4. Carry the load in a backpack or vehicle through the open world.
5. Reach an NPC buyer and sell it.
6. Spend the money on equipment, vehicles, plot leases, storage expansion, or
   end-of-Vakhta naming rights.

**1.2 Kolguyevite stays physical until it is sold.** Until it reaches a buyer it can be
carried, handed over, abandoned, stolen from a corpse, or taken from a destroyed
vehicle. It does not become money or leaderboard credit at extraction.

**1.3 Three principles.**

- Money, property, and competitive results are personal. There is no faction treasury.
- Mines produce nothing by themselves. A player must stand at the rock and work.
- Mine control grants access, not passive income, a faction reward, or a victory state.

**1.4 Configuration is the source of every number.** Prices, durations, yields,
capacities, fees, schedules, and limits live in server configuration. The PDFs never
substitute for live configuration: if a number affects an immediate decision, the
relevant interface must show it before the player commits.

## 2. The Vakhta

**2.1 Calendar.** A season is called a **Vakhta**. It contains a configured whole number
of Monday-to-Sunday weeks in server time. The calendar, current week, weekly boundary,
and planned wipe time are visible in game.

**2.2 Persistence.** The world persists through ordinary restarts during the Vakhta.
Balances, inventories, mine control, leases, installed plot modules, auctions,
registered vehicles, and other seasonal state are restored rather than recreated. The
mode has no automatic victory condition that ends the session.

**2.3 Beginning of a Vakhta.**

- Every mine begins neutral.
- Every regular player begins a new seasonal character with the configured opening
  balance, one-time starter loadout, and one starter cutter.
- The previous Vakhta's eligible carryover becomes part of that opening balance.
- Approved map names won at the previous endgame auction are applied for this Vakhta.

**2.4 Full wipe.** The wipe is an explicit, controlled operation. It removes all
gameplay state from the ending Vakhta, including:

- current balances, which are discarded; only previously recorded, capped crystal-sale
  carryover is applied to the next opening balance;
- carried equipment, offline character inventories, corpses, and world loot;
- secure storage contents and overflow;
- mine control, capture progress, alarms, and extraction cooldowns;
- plot leases, installed plot modules, bids, and storage-capacity bonuses;
- physical vehicles and cargo, registrations, access lists, restoration state, and
  garage state;
- current and completed in-game weekly leaderboards.

**2.5 What is not property.** An optional exported leaderboard archive may be kept by
the operators, but it is not progression and is not required for the next Vakhta.
Account permissions, bans, administrative audit records, the calculated opening
balance, and the next Vakhta's approved map-name assignments are operational records
rather than surviving property. Permanent authored plot-building shells are part of the
map rather than gameplay property, so a wipe leaves those empty shells in place.

## 3. Factions

**3.1 Faction matrix.**

| Faction | Mine access | Place in the world |
|---|---|---|
| **Meridian Group** | Captures mines; extracts only from Meridian-controlled mines | Corporate quarter and base in Ugoldar |
| **Granit Concern** | Captures mines; extracts only from Granit-controlled mines | Corporate quarter and base in Ugoldar |
| **Locals** | Cannot capture or own mines; extracts from neutral or controlled mines | Quarter in Ugoldar and a second spawn in Kamensk |
| **Garrison** | Does not participate in mine control or extraction | Whitelisted military faction on the southern island |

**3.2 Regular factions and the roster count.** Meridian, Granit, and the Locals are the
**regular factions**. Faction admission counts each authenticated player identity's
current regular-faction assignment in this Vakhta, including players who are offline. A
whitelisted player's Garrison character is not part of this count. Inactivity does not
free a place; changing regular faction moves that identity from one count to another.

**3.3 Balance limits.** The server evaluates the projected roster after every initial
selection or regular-faction change:

- Meridian or Granit may not exceed the other corporation by more than the configured
  maximum roster lead. That lead must be at least one so the first corporate player can
  be admitted.
- The projected Local count may not exceed the whole-number capacity obtained by
  rounding down the configured share of the projected total regular roster. This is a
  hard maximum, not a target. The launch value is 20%, so without a bootstrap exception
  the first Local place opens only when the projected regular roster reaches five
  players.

**3.4 Admission is one operation.** Admission is one server-authoritative operation, so
simultaneous requests cannot both consume the same legal place. Lowering a configured
limit never evicts an assigned player, but further admissions to an over-limit faction
are refused until its projected roster is legal again. The selection screen shows
current counts, configured limits, and the specific reason for any refusal. For a
faction change, this admission check happens before the destructive confirmation; a
refused change removes nothing.

**3.5 Invitations.** An assigned regular player may send a lightweight invitation to one
authenticated, online player who holds no regular-faction assignment. Acceptance
requests the inviter's current faction and runs the same admission rules at that
moment. An invitation never reserves a place, enters a queue, or bypasses either
balance limit. A sender and a recipient may each have exactly one outstanding
invitation. Its lifetime and send cooldown are configured, and it ends on disconnect or
faction assignment.

**3.6 Meridian and Granit.** The two corporations have symmetric mechanical capability.
A corporate player cannot extract from a neutral mine or one controlled by the rival
company; the corporation must capture it first. This restriction applies only to
extraction. Corporate players may still ambush one another, take kolguyevite and
equipment through combat, destroy hostile vehicles, and sell whatever they physically
recover.

**3.7 The Locals.** The Locals never become a mine owner. They may work any neutral or
controlled mine at its normal yield. Cutting at a controlled mine raises an alarm for
the owning corporation. Their special advantage is access plus information, not a
private mine, production bonus, or invisible quota. On every ordinary respawn, a Local
may choose either the Local quarter in Ugoldar or the Local spawn in Kamensk. This
two-city choice is intentional; plot ownership does not add another spawn at launch.

**3.8 The Garrison is whitelisted.** Garrison access is granted by the administration
through a whitelist; nobody else can select the Garrison. Membership does **not** grant
access to the admin cockpit and does not imply that the player is an administrator,
moderator, or developer.

**3.9 Two characters for a whitelisted identity.** A whitelisted identity may hold
**two characters** in the same Vakhta: one Garrison character and, if it also passes
ordinary admission, one regular-faction character. Every rule treats them as two
separate people:

- the regular character is counted, admitted, changed, scored, and wiped under the
  regular rules as if the Garrison character did not exist;
- the Garrison character has its own equipment and its own wallet, which opens at zero;
  value moves between the two only through the ordinary logged transfer and physical
  handoff paths that apply between any two players;
- switching characters happens at the selection screen. The character left behind
  follows the ordinary departure rules of section 6: outside a protected city, or while
  holding Aggressor status, it stays in the world as a retained body for the same
  deadline as a disconnect, so a swap is never an escape from a fight;
- revoking the whitelist removes the Garrison character and its equipment and leaves
  the regular character untouched; a mid-Vakhta grant creates the Garrison character
  without touching the regular one.

**3.10 How the Garrison plays.** The Garrison follows the same full-loot, safe-zone,
aggressor, death, and disconnect rules as everybody else. It may move and fight
anywhere in the world without a formal event schedule. The island restriction applies
only in the other direction: a regular player entering the Garrison's southern island
receives a boundary warning and a configured grace countdown, and is killed if still
inside when it expires. Leaving the restricted territory in time cancels the countdown.

**3.11 Garrison equipment.** The faction has unlimited access to its operational
equipment and therefore receives no mining progression, weekly result, carryover, plot
lease, or naming-auction eligibility. Its equipment carries no special provenance tag:
gear genuinely taken from a defeated Garrison player is ordinary gear and may be kept or
sold. Withdrawing gear to sell it directly, manufacture money, or feed another faction,
including the player's own regular character, is abuse of the whitelist and is handled
through logs and bans rather than a second item economy.

**3.12 Changing a regular faction.** Changing between Meridian, Granit, and the Locals
starts a fresh character for the current Vakhta. It may be requested only by a living,
on-foot, unmarked player at the authored faction-administration service in Ugoldar. The
request is refused from the field, from a vehicle, or while Aggressor status is active.
The player confirms the consequences before the change:

- current balance, equipment, secure storage, and overflow are removed;
- current carryover progress is cleared;
- every Open or Paused bid is withdrawn and its reservation released. The change is
  refused while any bid is Locked or settling;
- current and already-awarded future plot rights end, and their installed Storage
  Expansions are removed without module compensation or lease refund;
- vehicles and registrations end. The change is refused while any registered vehicle
  is occupied; after confirmation, cargo in an unoccupied deployed vehicle is not
  recovered;
- the once-per-Vakhta starter loadout and cutter are not issued again;
- committed crystal-sale records remain as historical facts, including sales in the
  current week, and keep the faction recorded at the time of each sale. Those sales
  continue to appear in the personal board and faction-at-sale statistic;
- a naming assignment from an already-settled auction remains valid. It is a completed
  personal purchase, not an active faction entitlement.

**3.13 Transfers before a switch.** Transfers made before a switch remain in the
transaction log. Suspicious value movement is a cockpit review and moderation problem,
not a hidden transfer-graph formula.

## 4. The World

Kolguyev has two protected cities, contested countryside, old mine workings in the west
and centre, and the Garrison's separate island to the south.

**4.1 Ugoldar.** Ugoldar is the shared home city. It contains named Meridian, Granit,
and Local quarters. The streets and the quarters themselves are physically open: nobody
is stopped at an invisible faction boundary. What is restricted is service. A visitor
may walk through another faction's quarter, but cannot use its buyer, shop,
secure-storage terminal, garage, or faction-only plot. The only cross-faction garage
exception is the narrow **Return** action for a vehicle whose registration access list
names that visitor; it grants no shopping, deployment, restoration, or access-list
control. Meridian and Granit respawn at their bases here. The Locals may also choose
their Ugoldar quarter.

**4.2 Kamensk.** Kamensk is a neutral protected market in the north-east. It has no
faction territory. Its buyers pay more than the safe Ugoldar baseline, its catalog is
broader, and its plots are open across regular factions. Reaching it with a load
requires the journey across the island. There is no default personal storage in
Kamensk. A player who holds a Kamensk plot and installs its Storage Expansion gains
another access point into the same secure storage through that plot's console. Kamensk
is also a Local respawn choice, but a plot never becomes a spawn point at launch.

**4.3 Beyond the cities.** Everywhere outside Ugoldar and Kamensk is open full-PvP
ground. The Garrison island is the only faction-gated territory. Static cities,
safe-zone boundaries, mines, markets, garages, and other essential services are marked
on the map.

**4.4 Travel must create encounters.** The world is laid out so that:

- the Ugoldar–Kamensk route is a major traffic spine through contested ground;
- easy workings near safe ground are less productive, while remote and exposed
  workings produce more per cutting cycle;
- Kamensk has no conveniently adjacent mine;
- at least one lower-yield mine is reachable from Ugoldar by a new player on foot;
- both protected cities need several dispersed exits with covered approaches, not one
  gate a single squad can seal.

These are binding map-design constraints, not promises of exact distances or mine
counts. The actual sites and values are configured and verified on the shipped terrain.

## 5. Your Character

**5.1 Starter issue.** On the first spawn of a Vakhta, a regular player receives the
configured starter loadout and one starter cutter. Each is issued once for that player
identity during that Vakhta. Death, poverty, faction respawn, reconnecting, and losing
the cutter do not issue another copy.

**5.2 The cutter.** The cutter is ordinary physical equipment after issue. It does not
wear out through use, but it can be carried, dropped, sold, stolen, or lost with the
rest of a character's equipment. A replacement must be bought or obtained from another
player.

**5.3 Death and loot.**

- Personal money is virtual and survives ordinary death.
- Eligible equipment already in secure storage survives ordinary death.
- Everything carried by the character becomes part of an ordinary lootable corpse,
  including equipment, a cutter, and kolguyevite.
- A corpse remains lootable inside a protected city. The safe zone prevents PvP damage;
  it does not turn an existing corpse into private storage. Any dead occupant may be
  removed from a vehicle despite its access list.

**5.4 Respawn.** Meridian and Granit respawn at their Ugoldar bases. Locals choose
Ugoldar or Kamensk. The Garrison respawns at its own base. There is no plot respawn at
launch and no paid forward respawn attached to mine control.

## 6. Protected Cities, Aggression, and Leaving the Server

**6.1 Safe-zone protection.** Ugoldar and Kamensk reject all non-self player-attributed
damage whenever either the protected target or the attack's source or fire origin is
inside the city. A projectile launched from protected ground remains blocked after
crossing the boundary. For a remote explosive, the controlling player's position when
it is triggered is also an attack source. Player-driven impact damage to another
protected character, occupied vehicle, or registered vehicle is rejected as
player-attributed damage. If that impact cannot be attributed reliably, the server
conservatively rejects collision damage to those protected targets. Weapon holstering
and disabled actions are useful presentation, but they are not the authority that makes
the city safe. The Garrison follows the same rule.

**6.2 What the safe zone does not do.** It does not suppress non-player-attributed
environmental damage. Registered-vehicle access also remains in force inside a city.

**6.3 What gives Aggressor status.** The following actions give the acting player
**Aggressor** status:

- firing a weapon, even if nothing is hit;
- releasing, triggering, or remotely detonating a grenade, explosive, or another
  offensive throwable;
- dealing player-attributed damage to any other player, including a faction ally;
- dealing player-attributed damage to another player's occupied or registered vehicle.

**6.4 Collisions.** For vehicle collisions, the damage bullets above apply only when the
server accepts actual positive damage at or above the configured collision threshold
and can reliably identify the player who was driving at the moment of contact. An
ambiguous collision marks nobody. This narrow Aggressor rule does not weaken the
conservative safe-zone veto: uncertain collision damage to a protected target is still
rejected.

**6.5 What does not mark a player.** Receiving damage alone does not mark the victim.
Returning fire does mark the defender under the same rules as any other attacker, so
that player cannot then enter a protected city until the status expires or death clears
it. Self-inflicted and environmental damage do not create the status. The duration is
configured, is intended to be long enough to finish an interception near a city, and is
shown continuously to the marked player.

**6.6 City enforcement.** A marked aggressor who enters a protected city is killed
immediately by the city's server-side enforcement. A player who becomes marked while
already inside is killed by the same rule. Until death or expiry, the status also
rejects trader, secure-storage, garage, plot-console, and clean-logout operations.
Death clears it.

**6.7 Executing a driver.** Executing a driver does not stop, unlock, confiscate, or
empty the vehicle. It continues under ordinary vehicle physics, and its registration
access list, cargo, damage, and safe-zone rules remain unchanged. A vehicle may
therefore coast across the boundary after its aggressor driver dies. This accepted
consequence keeps enforcement on the player rather than inventing a special vehicle
seizure rule.

**6.8 What the boundary protects.** The boundary shelters a living target from a marked
attacker: the attacker cannot personally cross it alive and continue the fight. It does
not prevent the accepted vehicle-and-cargo consequence above.

**6.9 Leaving from inside a city.** A character in a protected city with no Aggressor
status is saved and removed immediately; no disconnect body is retained there.

**6.10 Leaving from anywhere else.** Every other non-shutdown departure follows the same
rule, whether caused by Quit, connection loss, a client crash, or a kick:

1. The same character remains in the world for the configured base period, whose launch
   value is five minutes.
2. It remains killable and lootable.
3. Reconnecting in time returns the player to that same living character.
4. If the character dies, normal corpse rules apply.
5. If the applicable deadline expires while the character is alive, it is saved and
   removed.

**6.11 The retained-body deadline.** The deadline is whichever is later: the departure
time plus the configured base period, or the current Aggressor expiry. Disconnecting
must never shorten the consequence of attacking; an Aggressor refresh while the body
remains also moves the deadline later. If a living character is saved and removed when
that deadline passes, a later reconnect resumes the same saved character at its saved
position.

**6.12 Seats and shutdowns.** A disconnected character stays in its vehicle seat.
Ordinary dead-occupant removal keeps the body lootable even if the vehicle's access
list would otherwise reject the looter. Planned server shutdowns save characters
normally.

**6.13 Kolguyevite on logout.** Kolguyevite has no special logout rule. It remains in
the character's ordinary inventory and follows the character through the process
above.

## 7. Personal Economy and Recovery

**7.1 Money.** Money is personal, virtual, and not lootable. It survives ordinary death
and is removed by a full wipe or a confirmed regular-faction change. There is no
faction treasury, automatic squad payroll, or pay-for-presence system.

**7.2 Equipment.** Weapons, tools, repair supplies, and ordinary equipment are physical
and full-loot. NPC catalog stock is not depleted by other customers. Buy and resale
prices must always prevent buying from an NPC and immediately selling back for profit.

**7.3 The city maintenance run.** A regular player whose total wallet balance,
including money reserved in bids, is below the configured recovery threshold can take a
**city maintenance run** in Ugoldar or Kamensk. Equipment, vehicles, and other owned
property are deliberately ignored by this simple cash test. The player accepts a work
order, visits the configured service points inside that same protected city, and
completes the shown maintenance interaction at each point. It requires no purchased
mining tool or personal vehicle and pays a modest configured amount of personal money
at the final point. The interface shows the route, payout, eligibility, and any cooldown
before acceptance. The Garrison is ineligible.

**7.4 Why odd jobs exist.** Odd jobs exist so a ruined player can buy an ordinary kit
and return to the island. They are deliberately less rewarding than successful mining
or robbery. Their payouts create neither weekly leaderboard credit nor carryover.

**7.5 The recovery threshold.** The threshold must be configured at or above the full
NPC purchase price of the single cutter available at launch. This keeps an eligible
player on the recovery path until buying that cutter is financially possible; there are
no cutter tiers to compare. There is no repeatedly issued poverty cutter or endlessly
renewed free combat kit.

**7.6 Money transfers.** Players may transfer money directly to another identified
player. The server validates the balance and records both sides of the transfer.
Transfers create no crystal-sale result and no carryover.

**7.7 Transfers involving the Garrison.** Transfers involving the Garrison, including a
Garrison-to-regular transfer and a transfer between a whitelisted player's own two
characters, are allowed by the same mechanic and recorded for whitelist-abuse review. A
transfer does not make direct arsenal liquidation or feeding another faction
legitimate; those remain the logged moderation violations described in 3.11.

**7.8 Items change hands physically.** A player hands over, drops, or exposes the item
and another player takes it. The launch version has no atomic items-for-money trade
window and no split-on-sale interface. A seller handling a squad's load uses ordinary
transfers if they distribute money afterward.

**7.9 Deals and betrayals.** Deals and betrayals performed through these mechanics are
part of player commerce. Operators correct proven server faults, not ordinary bad
bargains; exploits, real-money trading, account abuse, and administrator impersonation
remain moderation matters.

## 8. Markets and Traders

**8.1 Trader matrix.**

| Location | Who may use it | NPC purchase price | Player purchase catalog |
|---|---|---|---|
| **Meridian home traders, Ugoldar** | Meridian | Safe baseline for kolguyevite and eligible used gear | Meridian corporate equipment and goods |
| **Granit home traders, Ugoldar** | Granit | Safe baseline for kolguyevite and eligible used gear | Granit corporate equipment and goods |
| **Local fence, Ugoldar** | Locals | The same safe baseline | Local-exclusive equipment and goods |
| **Kamensk market** | Every regular faction; Garrison may sell eligible gear only | The same higher price for regular-faction crystal and used gear; eligible Garrison gear uses the gear quote | Broader black-market selection for regular factions |

**8.2 Channels and prices.** For regular factions, all four channels buy both kolguyevite
and eligible ordinary battlefield equipment. Kamensk's higher price is payment for
bringing value across dangerous ground. The Local fence gives the Locals a safe buyer at
home and their own catalog, but no special crystal premium. Meridian and Granit
therefore gain nothing by passing a load through a friendly Local, and Locals receive
the same Kamensk premium as everybody else.

**8.3 The Garrison at Kamensk.** The Garrison may use Kamensk only to sell eligible
ordinary gear. It cannot sell kolguyevite there, buy from the regular catalog, receive a
weekly result, or earn carryover. Direct arsenal resale or feeding remains whitelist
abuse even though the ordinary gear transaction is technically possible and logged.

**8.4 No provenance.** There is no corporate, stolen, army-pattern, previous-owner,
mine, or faction stamp on ordinary gear. A quote depends on the item, buyer, and
current configuration, not on who once carried it. The Kamensk catalog is configured
rather than dynamically rebuilt from items sold to the market.

**8.5 Before the sale.** Before confirmation, a buyer shows access, accepted items,
quantities, unit prices, and total proceeds. A crystal buyer can sell all eligible
carried kolguyevite in one server-authoritative transaction.

**8.6 The receipt.** A crystal-sale receipt shows:

- what and how much was sold;
- the applicable price and total payment;
- the resulting weekly crystal-sale quantity;
- the carryover earned by this sale and progress toward the current weekly cap.

A gear-sale receipt explicitly shows that it earned no carryover. Every refusal gives
the real reason, such as wrong faction, no eligible item, invalid price, or Aggressor
status.

## 9. Mines and Control

**9.1 Mine states.** Each mine has a stable state, **Neutral**, **Meridian**, or
**Granit**, and may also have capture progress. Ownership and active capture are public
on the map. Mines begin neutral after a wipe.

**9.2 Who counts for capture.** Only living, connected Meridian and Granit players count
for capture. A retained body whose player has disconnected does not count. The Locals
and Garrison can fight inside the zone but do not move or mechanically contest its
progress.

**9.3 Capture progress.**

- When one corporation has a numerical advantage inside the capture zone, progress
  moves toward that corporation at the configured rate.
- Equal corporate presence pauses progress.
- With no corporation present, incomplete progress returns toward the mine's current
  stable owner, or toward Neutral if the mine has no owner.
- A completed transition sets the new owner and notifies players in the workings.

**9.4 Fixed configuration, no lock.** Capture is fixed by ordinary configuration, not an
Activity Index or hidden server-population formula. There is no post-capture lock. The
mine can be used immediately by players who now meet its access rule.

**9.5 Ownership checks during a cycle.** An extraction cycle checks ownership both when
work begins and when output is awarded. A control change does not confiscate crystal
already in a backpack, corpse, vehicle, or on the ground.

**9.6 Why control matters.** For Meridian and Granit, control is the right to extract.
It denies the rival company that same right and gives the owner a warning after the
first completed Local cutting cycle there. It does not generate passive money, modify
trader prices, create a faction pool, or produce an automatic defence payment.

**9.7 Why some mines are better.** Each mine has a configured **yield per completed
cutting cycle**. Easy sites near safe ground produce less; remote or exposed sites
produce more. A better yield fills the same carrying capacity in fewer attended cycles.
It does not create a different crystal grade or a more valuable item after extraction.
The map and the extraction prompt show the current yield. Placement and tuning must
make the saved working time worth considering against a longer, more dangerous haul;
otherwise the remote mine has no practical advantage.

## 10. Cutting Kolguyevite

**10.1 Nothing accumulates.** Kolguyevite is cut manually at designer-authored working
faces. Nothing accumulates at a mine while players are absent, players build nothing
there, and there is no mine vault, hopper, rig, or shared inventory.

**10.2 The cutting action.**

- The player must carry a cutter and have permission to extract at that mine.
- Starting a cycle occupies the player and prevents ordinary movement and weapon use.
- Cutting is audible and visible to nearby players.
- Cancelling or interrupting the action produces nothing.
- Before work begins and again before completion, the server checks capacity for the
  configured output.
- A full inventory prevents the action or completion with an explicit reason; output is
  never silently deleted or dropped.
- A successful cycle places the configured quantity directly into the player's ordinary
  inventory.

**10.3 Working faces.** Faces never deplete globally and are never reserved for one
squad. After a player finishes at a face, that face is unavailable to **that player
only** for its configured cooldown. Another player may use the same face immediately,
including at the same time. Dense groups of faces make the cooldown require a short
change of position rather than a long walk. The purpose is attended interaction and
movement, not limiting how many people a mine can support.

**10.4 Local cutting and the alarm.** A Local receives the mine's ordinary yield and
uses the same action. At a controlled mine, the first completed Local cycle starts an
owner-only alarm; later completed Local cycles refresh its configured lifetime.
Notifications may be throttled, but the alarm state uses one continuous clock. It
identifies the mine, never the player, squad, exact headcount, or carried amount. A
neutral mine has no owner to alert.

**10.5 The alarm is information only.** It does not reduce prices, create a defence
contract, limit the Local's extraction, or change the mine's yield.

**10.6 Local mine-activity intelligence.** The Locals receive a separate coarse view of
recent cutting activity by mine. It is delayed by a configured amount and reports only
broad activity buckets. It does not show names, factions, exact player counts, routes,
inventories, or vehicle contents.

**10.7 Why the intelligence exists.** This information is the Locals' main structural
advantage: it gives a raider somewhere plausible to look without guaranteeing a target.
Mine owner and configured yield remain public to everyone; the delayed activity layer
is Local-only.

## 11. Kolguyevite and the Haul

**11.1 One item, no history.** Kolguyevite is one quantity-bearing inventory item type.
It carries no origin faction, mine, owner, extraction time, week, purity, or grade.
Units from different places are interchangeable and may be combined without changing
their meaning. Its weight and occupied capacity follow the amount carried.

**11.2 Where it may exist.**

- a character inventory or backpack;
- vehicle cargo;
- a corpse;
- a temporary public loot container;
- the open world.

**11.3 Where it is rejected.** It is rejected from every secure-storage path, including
the home locker, a plot terminal, a nested bag moved into secure storage, or an attempt
to put crystal into a bag already stored there. A disabled client action is only
feedback; the server owns the final decision.

**11.4 Vehicles and saved characters.** Vehicle cargo is explicitly legal, including
while the vehicle is inside a protected city. A legitimately saved character may also
leave the server while carrying it. Neither location has a crystal-specific expiry,
ejection, or forced-drop rule. Vehicle destruction, abandonment cleanup, character
exposure during an unsafe disconnect, the weekly sale clock, and the full wipe still
apply normally.

**11.5 Whoever sells, scores.** Whoever sells kolguyevite to an NPC receives the money
and the personal weekly result. Because the item has no history, the system does not
distinguish mining, theft, a gift, or a squad handoff, and a load held across a weekly
boundary counts when it is eventually sold. This simplicity is intentional.

## 12. Registered Vehicles

**12.1 Scope.** Only vehicles bought through an eligible garage catalog use the
registration system. The catalog is a whitelist of approved vehicle definitions; an
ambient world vehicle that looks similar does not silently become registered.

**12.2 Registrations and the deployed limit.** Buying an eligible vehicle creates a
registration for the current Vakhta. A player may own several registrations. A separate
configured limit controls how many of that player's registered vehicles may be deployed
at the same time. If that limit is lowered below the player's current deployed count,
existing vehicles remain until return, loss, or wipe, but no additional deployment is
accepted while the count is at or above the new limit.

**12.3 Registration states.**

| State | Meaning |
|---|---|
| **In garage** | No functional instance exists; it may be deployed for free if the owner is below the active limit and selects a valid clear position in the garage's deployment area. |
| **In world** | The one functional instance exists somewhere in the world; the garage cannot create another. |
| **Lost** | Destruction or confirmed abandonment cleanup ended the functional instance; restoration is required. |

**12.4 Wrecks.** A wreck may remain until normal garbage collection, but an inert wreck
is not a functional vehicle and does not block restoration.

**12.5 Deployment.** Deployment uses a Conflict-style placement preview inside the
garage's authored area. Green or red presentation gives immediate guidance, but the
server rechecks ownership, the deployed limit, registration state, allowed vehicle
definition, terrain, overlap, and clearance when the owner confirms. A refused
placement creates no vehicle, leaves the registration In garage, and charges nothing.
The player must choose another valid position or wait for the obstruction to move; the
garage does not tow or automatically return somebody else's vehicle to clear space.

**12.6 Finding the vehicle.** The owner receives a marker for each registered vehicle
currently in the world. Being killed away from it does not create a replacement: the
owner must travel back, arrange a ride, or wait for ordinary abandonment cleanup.

**12.7 The access list.** Every registration has one owner-managed **access list**:

- the owner is always authorized;
- named authorized players may use every seat and remove cargo;
- an authorized player may physically return the vehicle to its garage, while only the
  owner may deploy it, restore a Lost registration, or edit the access list;
- faction membership alone grants no access;
- there is no PIN, shared code, or driver/passenger/cargo permission matrix;
- the list remains attached to the registration through garage return, destruction,
  and restoration during the Vakhta.

**12.8 Unauthorized players.** Unauthorized players cannot enter, change into a seat, or
remove cargo. Outside a safe zone they may still damage and destroy the vehicle.
Destruction turns the contents into a public prize rather than making a locked truck
invulnerable.

**12.9 Returning.** A live vehicle uses the ordinary repair system. It may return to the
garage only when the actual vehicle is present at the terminal, empty, unoccupied, and
fully repaired. Garage return is not remote recovery or free repair.

**12.10 Destruction and restoration.** On destruction, remaining top-level cargo is
transferred exactly once into a temporary public loot crate at the wreck. Nested
contents stay in their containers. The registration enters Lost state, and the owner
pays the configured restoration fee to make it available in the garage again.

**12.11 Abandonment cleanup.** When normal abandonment cleanup removes a live
registered vehicle, its registration also becomes Lost. Cargo removed by cleanup is not
returned to the owner or garage. Cleanup uses the world's ordinary relevance and
nearby-player rules rather than a separate absolute lifetime for registered vehicles. A
vehicle, and any legal cargo in it, may therefore remain for longer while nearby players
keep it world-relevant, including inside a protected city. It still occupies one of the
owner's deployed slots and is removed by the full wipe. The current cleanup policy and
restoration cost are shown at the garage.

**12.12 What does not exist.** There is no remote recall, vehicle sale, retirement
refund, insurance tier, tracking beacon beyond the private owner marker,
shareable/public tracker, or theft code. A full wipe removes registrations, access
lists, physical vehicles, cargo, and all garage/restoration state.

## 13. Personal Secure Storage

**13.1 One pool.** Each regular player has one logical, server-owned secure-storage
pool. A home locker and the storage action on every plot console enabled by that
player's active Storage Expansion are access points into the same pool; they are not
separate inventories and do not mirror physical containers.

**13.2 Cross-city access.** Eligible ordinary gear deposited through a Kamensk plot
console can be withdrawn through the home terminal in Ugoldar, and vice versa. This is
an intentional consequence. Kolguyevite is still rejected, so the shared pool cannot
move a crystal load between cities.

**13.3 Capacity.** The effective capacity is the configured base capacity plus the
configured bonus from every active Storage Expansion. One Storage Expansion may be
installed on each held plot, and bonuses from several held plots stack. The interface
shows base capacity, each active plot bonus, current use, and any overflow.

**13.4 Access checks.** Only the owner can access the contents, and every mutation
rechecks identity, terminal access, Aggressor status, capacity, and the crystal
restriction.

**13.5 Overflow.** When a lease or its Storage Expansion ends, that module's capacity
bonus is removed immediately. Stored items are not moved, dropped, or deleted. If use
now exceeds the reduced capacity:

- existing items remain visible and may be withdrawn;
- no deposit is accepted through any terminal while the pool is over capacity;
- deposits resume after withdrawals bring use within the current limit.

Regaining capacity clears the restriction automatically.

**13.6 Death, change, wipe.** Ordinary death does not affect the pool. A regular-faction
change or full wipe removes both normal and overflow contents.

## 14. Plots and Storage Expansion

**14.1 What a plot is.** Plots are authored properties inside protected cities. Every
plot contains a permanent, map-authored building shell that remains unchanged between
holders, one management console, and one fixed module position. The shell and its
interior remain public and stay in the world whether the plot is leased or vacant; a
lease grants management rights rather than ownership of the building. Plot spawning is
not part of the launch system.

**14.2 Eligibility and limits.**

- **Ugoldar plots** belong to a named faction quarter. Only a member of that regular
  faction may bid for and hold one.
- **Kamensk plots** are neutral. Any regular faction may bid for them.
- The configured personal lease limit applies across both cities.
- A configurable Kamensk faction limit may be enabled to prevent one faction taking the
  entire district.

**14.3 Lowered limits.** If either lease limit is lowered, current leases and
already-settled future leases are grandfathered through their awarded terms. New bids
use the lowered limit against the projected post-transition holdings, so an excess
cannot be renewed or expanded; nothing is confiscated mid-term.

**14.4 Storage Expansion.** A bare lease adds no storage capacity or secure-storage
action at the plot console. The same console is the plot's sole interaction endpoint:
it always offers holder-only management and, at launch, its only purchase is one
**Storage Expansion**. The leaseholder buys it directly for the displayed configured
price; it is not an inventory item, build kit, freely placed structure, or separate
container. Installation adds the secure-storage action to that console, adds its
configured bonus to the player's one logical pool, and records the exact price paid.
The fixed module position is only the authored presentation and binding for this
purchase. A plot accepts only one installation, with no storage tiers, upgrades, or
second socket. Expansions on multiple active plots stack.

**14.5 No player construction.** The authored shell and module location prevent
player-built obstructions. Players cannot place, rotate, relocate, damage, dismantle, or
edit either one. Holding a plot never adds a death-screen spawn or replaces the
player's normal faction respawn.

**14.6 Lease price.** Lease duration and the auction calendar are configured. The
winning auction payment is the entire lease price; there is no additional recurring
rent.

**14.7 Lease transitions.**

- If the same player wins the next term for the same plot, the lease continues and the
  installed Storage Expansion remains active.
- If another player wins, or the plot receives no valid renewal, the old lease ends.
  Its Storage Expansion is removed and the former holder receives the configured
  percentage of the exact recorded purchase price.
- No item, build kit, or blueprint is returned. Terminal access and capacity update with
  the module; overflow follows rule 13.5.
- A confirmed regular-faction change removes the player's Storage Expansions without
  module compensation or lease refund.
- No lease or installed module crosses the full wipe, and wipe removal pays no
  compensation. The permanent authored building shell remains as map infrastructure.

**14.8 Future modules.** Additional module types are deferred until they receive a
separate design. The launch implementation does not create a generic upgrade tree for
hypothetical future modules.

## 15. One Auction System

**15.1 One engine.** Plot leases and next-Vakhta map naming use one server-authoritative
auction engine with two narrow reward types. It is a sealed, first-price auction with a
fixed closing time.

**15.2 Placing a bid.**

- A player may have one current exact bid per lot.
- The player may replace or withdraw it while the lot is open.
- The bid amount is reserved immediately and cannot be spent, transferred, or reused in
  another bid.
- Other bids and the current leader remain hidden before closing.
- A bid receipt shows the lot, the player's amount, the reserved balance, and the close
  time.

**15.3 Closing.** At closing, the highest valid bid wins and the winner pays the full
submitted amount. Equal bids are ordered by when the current bid was accepted by the
server; revising a bid gives it a new position. Losing, withdrawn, cancelled, and
invalid bids release their reservations automatically.

**15.4 What the auction does not have.** There is no minimum increment, proxy maximum,
soft-close extension, incumbent bonus, escalating occupancy premium, or attended
ceremony required for normal settlement. Minimum bid, schedule, eligibility, and all
limits are configured and shown before a bid.

**15.5 Plot auction limits.** An open plot bid reserves one future lease slot. A player
cannot hold or bid for more plots than the configured personal capacity. A lease that is
about to expire may support either a bid to renew that plot or a bid for another one,
not several simultaneous wins. Faction and Kamensk limits are checked both when the bid
is accepted and at settlement. Lots sharing a closing time settle in a published stable
order. If an earlier result makes the highest bid in a later lot ineligible, that later
lot falls to its next eligible bidder or remains unleased if none exists.

**15.6 Endgame naming rights.** The initial endgame reward is limited to configured map
labels, such as mines or plot districts, for the next Vakhta. It has no gameplay power.
Candidate names are submitted and moderated before bidding. A naming bid references one
candidate already approved for that player and lot. Approvals do not change while the
lot is open, so the winning name can be applied automatically. If no valid bid settles,
the canonical map name remains.

**15.7 Lifetime of a name.** Naming assignments survive only the wipe needed to apply
them to the next Vakhta and expire at its end. Monument plaques, permanent titles,
community-vote slots, and a separate prestige currency are not part of the launch
system.

**15.8 Interruptions.** Auctions, reservations, and settlement survive logout, death,
and ordinary restart. A healthy lot settles automatically. Recovery follows a published
deterministic rule: if the server accepted bids through the closing time, the lot
settles normally; otherwise operators extend or cancel it before bids are unsealed. A
lot is never reopened or cancelled in response to revealed bids. Every intervention is
recorded.

**15.9 Cancellation.** Cancellation is available only before a lot locks. Once locked,
its deterministic settlement must finish or be repaired. If an incident cancels a
plot-renewal lot, the current lease continues without additional rent through the
published close and settlement of its replacement lot; an already-unleased plot remains
unleased. Cancelling a naming lot leaves the canonical map name in place.

**15.10 Operator boundary.** Operators may schedule, pause, extend, cancel and refund an
eligible open lot, or repair a failed settlement. They cannot edit a player's bid or
choose a different winner in the normal process. Active bid identities and values
remain sealed until the lot locks.

## 16. Weekly Results and Carryover

**16.1 The personal weekly leaderboard.** Every Monday-to-Sunday week has one personal
leaderboard. It ranks regular players by the quantity of kolguyevite accepted from them
by NPC buyers during that week.

**16.2 Board rules.**

- The seller receives the result, regardless of who extracted or previously carried the
  item.
- The sale belongs to the week in which it commits.
- Unsold crystal may cross a week boundary and count in the later week.
- Equal totals share the same result; there is no hidden combat or revenue tiebreaker.
- The current week is visible live, and completed weeks remain available until the
  Vakhta wipe.
- Locals appear under the same player rules as Meridian and Granit. The Garrison is
  excluded.

**16.3 No other boards.** There is no kill, death, survival, efficiency, income, or
composite-rating leaderboard.

**16.4 Faction statistic.** A separate current-week widget sums the same crystal-sale
quantities by the seller's faction at the time of sale. It answers only which faction is
currently selling the most. It grants no reward, winner, territory, season point,
banner, or tiebreaker.

**16.5 Carryover into the next Vakhta.** Each direct NPC crystal sale also contributes a
configured share of its money proceeds toward the player's next opening balance, up to
that player's configured cap for the current week.

- Every week has its own cap.
- Reaching it stops further carryover credit for that week but does not reduce current
  sale proceeds.
- A new cap opens at the weekly boundary.
- Unused capacity never moves to another week.
- Gear sales, odd jobs, player transfers, auction refunds, plot compensation,
  administrative compensation, and any other payment create no carryover.

**16.6 Applying carryover.** The receipt shows the contribution and current cap
progress. At the full wipe, all weekly contributions are summed into the next opening
balance and the old contribution records are cleared after successful application.

**16.7 No season victory.** There is no faction victory, shipment boat, doubled final
week, or fresh-crystal surge. Leaderboard placement and the faction widget grant no
payout or automatic reward; next-Vakhta naming rights are purchased separately through
auctions, not awarded by season rank. Weekly activity remains relevant because each week
offers a new personal result and a new carryover cap.

## 17. Information, Receipts, and Administration

**17.1 What players can see.** The game exposes current values at the point of use
rather than asking players to find a possibly stale PDF. This includes:

- Vakhta dates, current week, weekly leaderboard, faction statistic, and carryover
  progress;
- faction eligibility, roster counts and limits, invitation outcome, and faction-change
  consequences;
- safe-zone boundaries, Aggressor status, its remaining duration, and logout warning;
- mine owner, capture progress, extraction permission, cycle yield, action time, and
  personal face cooldown;
- Local mine-activity intelligence and corporate Local-cutting alarms;
- market access, catalog, quotes, proceeds, and carryover effect;
- secure-storage capacity, each installed plot bonus, current use, and overflow;
- plot terms, Storage Expansion price and state, auction schedule, the player's own bid,
  and reserved money;
- vehicle registration state, deployed limit, placement validity, access list, location,
  cleanup policy, and restoration fee.

**17.2 Receipts.** Consequential server-mediated transactions produce receipts:
purchases and sales, transfers, jobs, bids and refunds, lease transitions, compensation,
and vehicle fees. Important results that settle while the player is offline are
available after reconnect. Every refused action states the real reason rather than
silently doing nothing.

**17.3 The operator cockpit.** The first Vakhtas are expected to require active manual
operation. A launch-critical cockpit therefore provides authorized staff with live
visibility and controlled actions for server health, players, transactions, Garrison
activity, mines, alarms, capture, disconnect bodies, vehicles, storage, plots, auctions,
weekly boundaries, configuration, backups, incidents, broadcasts, and the wipe.

**17.4 Cockpit authority.** Cockpit control does not mean arbitrary database editing.
Commands use the same server-authoritative domain operations as normal gameplay,
include state validation, require appropriate permissions, and append the acting
operator, reason, before/after state, time, and outcome to an application-append-only
audit trail retained off-host. Garrison membership grants none of these permissions.

**17.5 Routine and exceptional operations.** Routine authorized operations include
configuration, schedules, auctions, broadcasts, maintenance, and wipe workflows.
Player-value compensation or rollback requires a proven server fault and an explicit
compensating operation. Operators do not silently rewrite completed sales, transfers,
leaderboards, active bids, or auction winners. Destructive global operations require a
preview, verified backup, explicit confirmation, and resumable execution.

## 18. Launch Boundary

**18.1 Not in the launch version.** The systems above define the intended launch game.
The following mechanics are not in the launch version unless a later design decision
adds them:

- an Activity Index or population-scaled economy;
- faction victory, weekly boats, season banners, and finale multipliers;
- crystal provenance, mine grades, purity tiers, origin weeks, or holder stamps;
- Local yield penalties, per-window extraction quotas, and corporate extraction from
  neutral or another corporation's controlled mines;
- mine buyback bonuses, alarm price suppression, and defence contracts;
- an atomic player-trade window and split-on-sale payroll;
- dynamic pricing, story contracts, quests, and world events beyond recovery jobs;
- formal Garrison events, special combat immunity, or Garrison item provenance;
- vehicle PINs, public/shareable/additional trackers beyond the private owner marker,
  registered-vehicle theft, insurance, sale, remote recall, towing, or a custom absolute
  cleanup lifetime;
- plot spawning, free-form plot construction, storage tiers, additional plot modules,
  destructible plot raids, separate plot inventories, additional rent, proxy bidding,
  soft closes, or escalating incumbent premiums;
- a persistent faction-invitation queue, reserved invitation slots, or an invitation
  bypass around faction-balance limits;
- monuments, permanent titles, and a separate prestige system;
- an indefinite offline sleeper system.

**18.2 Accepted consequences.** Several consequences are accepted because removing them
would rebuild discarded complexity:

- a squad may nominate one seller for its crystal;
- a corporation may extract from a mine it does not control through an allied Local, at
  single-cutter throughput and under the owner alarm;
- crystal may be held on an offline character or in a locked safe-zone vehicle, and a
  relevant vehicle may persist in a busy protected city under ordinary world cleanup;
- killing an aggressor driver does not seize or stop the vehicle, so a vehicle and its
  cargo may coast into protection without that driver;
- a defender who returns fire becomes an Aggressor too;
- offline or inactive regular-faction assignments continue occupying roster capacity
  for the current Vakhta;
- Local two-city respawn can be used as death travel;
- shared plot storage moves ordinary gear between terminals;
- Garrison equipment and money transfers rely on whitelist enforcement and audit;
- physical player trade permits scams.

**18.3 Visible choices.** These are visible design choices, not missing rules.

## Appendix A. Binding World-Placement Checks

**A.1 Checks before a map revision ships.** Validate these relationships on the actual
terrain:

1. Mine count concentrates players at the expected population instead of scattering
   them; the count itself remains configurable through authored site activation.
2. Mines remain primarily in the west and centre, with yield increasing enough toward
   remote and exposed sites to compensate for some, but not all, of the travel risk.
3. A lower-yield site is reachable from Ugoldar by a new player on foot, but no working
   sits directly on a protected boundary.
4. Kamensk has no conveniently adjacent mine.
5. Working faces form dense groups so a personal face cooldown causes a short move.
6. Valuable routes intersect the contested centre and the Ugoldar–Kamensk traffic spine;
   a safe bypass must not erase the primary interception route.
7. Each protected city has several useful exits around its perimeter, covered approaches,
   and no protected firing position with a commanding sightline over an exit.
8. Authored plot shells, module positions, and interaction volumes do not overlap roads,
   city exits, neighbouring properties, or essential service volumes. Every plot
   console, garage, trader, and boundary is readable before interaction.
9. Each garage's authored deployment area offers multiple useful positions and enough
   clearance that a single parked vehicle cannot block every legal deployment.

**A.2 Method.** These checks are evaluated through a playable route test, not from map
coordinates alone.
