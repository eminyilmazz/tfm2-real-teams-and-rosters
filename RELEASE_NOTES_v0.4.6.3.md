# v0.4.6.3 - Generated Player Language Regions

This release packages the Stage47 database after in-game validation of the Stage46 player-limit build. It fixes generated/random visible players so their language region follows the team slot they play in instead of remaining stuck on the original generated-row language.

## Changelog

- Fixed generated/random player language vectors to match each visible team slot's game region.
- Rewrote 124 generated-player dynamic language vectors; 52 generated-player rows were already correct.
- Left real/current-core player language vectors unchanged.
- Preserved the load-tested random extra-player limiting from `v0.4.6.2`.
- Preserved the corrected LCP/APAC team layout, moved players/stats/roles/logos, all-custom-logo refs, sourced head coaches, balance pass, and NRG square logo fix.
- Included the player-photo Steam Workshop thumbnail and local attribution notes.

## Validation

- Packaged database SHA256: `0cafe658d6e7cd624c1bc29c3d1cd40987ccdeb585e3f6b9e08d1cdfd4a202bb`
- Import package kind: `1`
- Gzip offset: `25`
- Parsed teams: `120`
- Embedded custom logo PNG blocks: `120`
- Teams using custom logo refs: `120`
- Teams using default logo refs: `0`
- Generated/random visible players checked: `176`
- Generated/random language vectors changed: `124`
- Semantic diff versus Stage46: `0` athlete non-language diffs, `0` team diffs, `0` staff diffs.
- Strict release validator passed with the known large zero-run warning.

## Notes

- The language fix uses the dynamic language-vector layout; no fixed-offset language writes are used.
- `database_pack.info` descriptions were not changed. Only version metadata is updated.