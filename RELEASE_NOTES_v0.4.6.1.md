# v0.4.6.1 - Player Origins, Balance, and Head Coaches

This release is the first mod update on the Teamfight Manager 2 `0.4.6` compatibility line and includes the latest roster-data improvements on top of the validated logo/roster package.

## Changelog

- Updated the packaged database for Teamfight Manager 2 `0.4.6` compatibility.
- Added player origin/communication-region fixes for transferred major-league players so scouting/player origin displays better match real rosters.
- Rebalanced player and league strength using the latest multi-year Oracle's Elixir base plus Games of Legends player/team adjustments.
- Improved real-world gaps between major top teams, lower major teams, minor leagues, division 2 teams, and regional leagues.
- Added sourced real-life staff head coaches for 89 teams by patching the displayed first staff row for each covered team.
- Included confirmed head-coach corrections such as Dplus KIA `cvMax`, Los Ratones `YamatoCannon`, and the user-checked SA Div 2 staff names.
- Left unresolved or ambiguous head-coach rows unchanged instead of guessing from broad search matches.
- Preserved the v0.4.4.4 roster display corrections and NRG square logo fix.

## Validation

- Packaged database SHA256: `6ced3e64363587eb3845dbc727c43b4f1014dd8db4fa92603668f5aa7ab70bec`
- Import package kind: `1`
- Gzip offset: `25`
- Parsed teams: `120`
- Embedded custom logo PNG blocks: `120`
- Teams using default logo refs: `2`
  - `Deep Cross Gaming`
  - `VARREL YOUTH`
- Teams using custom logo refs: `118`
- NRG logo ref remains custom and the embedded logo block remains present.
- Staff records scanned before/after the head-coach pass: `242`
- Staff names changed: `89`
- Parsed team semantic diffs versus the balanced source DB: `0`
- Parsed version-5 athlete semantic diffs versus the balanced source DB: `0`
- Strict release validator passed with only the known Deep Cross Gaming / VARREL YOUTH default-logo warnings and the known large zero-run warning.

## Notes

- `database_pack.info` is intentionally not changed in this release.
- Head-coach coverage is partial by design. Search-resolved pages and ambiguous multi-head-coach sources were excluded unless manually confirmed.
- The remaining unresolved/review-only head-coach rows are tracked in the Stage30 audit files in the working data repo.
