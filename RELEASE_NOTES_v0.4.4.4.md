# v0.4.4.4 - EU Roster Ownership Hotfix

Built for Teamfight Manager 2 v0.4.4.

This release keeps the mod versioning aligned with the game version: `0.4.4` for the current game build, plus `.4` for the fourth mod patch on top of it.

## Why This Hotfix Exists

The v0.4.4.3 package fixed the direct-import logo wrapper, but it was built from a database where the EU team rows had been corrected after an older athlete pass was already in place. That left several EU teams with the previous slot's roster.

Reported examples were confirmed:

- GIANTX was carrying Karmine Corp Blue players.
- Fnatic was carrying GIANTX players.
- SK Gaming was carrying Fnatic players.
- Team Heretics was carrying the old Shifters roster.

## Fix

- Regenerated the athlete pass from the current 120-slot manifest instead of applying team-row fixes on top of stale athlete ownership.
- Kept the already-correct v0.4.4.3 team rows and logo refs intact.
- Rebuilt the direct-import package from the verified logo import template.
- Did not include the rolled-back stadium-name experiment.

## Corrected EU Slot Check

The corrected package now exports these starter assignments:

- Slot 25, GIANTX: Lot, ISMA, Jackies, Noah, Jun
- Slot 26, Fnatic: Empyros, Razork, Vladi, Upset, Lospa
- Slot 27, Los Ratones: Baus, Velja, Nemesis, Crownie, Rekkles
- Slot 28, SK Gaming: Wunder, Skeanz, LIDER, Jopa, Mikyx
- Slot 29, Team Heretics: Tracyn, Daglas, Serin, Ice, Way

## Preserved From v0.4.4.3

- Direct-import TFM2DB wrapper: `kind=1`, gzip offset `25`.
- All 120 team rows and logo references.
- All 120 embedded custom logo blocks.
- SillySilly Gaming remains on `custom:custom_team_logo/79`.
- Stadium names remain unchanged/defaulted.

## Validation

- Packaged asset: `tfm2_teams_and_rosters.tfm2db`
- SHA256: `0f3820793eea307dfd304474d495afffaafd6ddc86a7fede14e78caf746258da`
- File size: `32649858` bytes
- `tools/validate.go` passed on the packaged asset.
- Header verified as direct-import format: `kind=1`, gzip offset `25`.
- Re-export found 120 team logo references.
- Re-export found 120 embedded custom logo blocks.
- Team rows compared against v0.4.4.3 with 0 team text/logo/stadium/manager diffs.
- Embedded custom logo payloads compared against v0.4.4.3 with 0 block diffs.
- Generated starter roster check passed with 0 missing starters under exported team slots.

## Install

Download `tfm2_teams_and_rosters.tfm2db` and import it through the game's custom database/import flow.
