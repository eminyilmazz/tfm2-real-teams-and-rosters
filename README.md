# TFM2 Real-World Rosters 2026

A custom roster mod for **Teamfight Manager 2** featuring real League of Legends esports players, teams, and logos from the 2026 competitive season.

![Version](https://img.shields.io/badge/version-v0.4.4.3-blue)
![TFM2 Version](https://img.shields.io/badge/TFM2-v0.4.4-green)
![License](https://img.shields.io/badge/license-MIT-lightgrey)

## What This Mod Does

This mod replaces the default TFM2 roster database with real-world data:

- **🏆 120 Teams** - Real esports organizations from LCK, LPL, LEC, LCS, CBLOL, LLA, PCS, LJL, VCS, and more
- **👤 1,197 Players** - Real pro players with their in-game names
- **📊 Realistic Stats** - Player statistics derived from Oracle's Elixir and Games of Legends public data
- **🎨 Team Logos** - Embedded custom logos for major esports organizations

### Leagues Included

| Region | League | Teams |
|--------|--------|-------|
| Korea | LCK | Gen.G, T1, HLE, DK, KT, BNK, DRX, HJ, NS, DNS |
| China | LPL | BLG, JDG, TES, EDG, WBG, IG, LNG, WE, OMG, LGD |
| Europe | LEC | G2, FNC, VIT, KC, GX, NAVI, LR, KOI |
| North America | LCS | C9, TL, FLY, NRG, DIG, SEN, SR |
| Brazil | CBLOL | LOUD, paiN, FURIA, RED |
| And more... | | |

## Installation

### Quick Import (Recommended)

1. Download `tfm2_teams_and_rosters.tfm2db` from the [Releases](../../releases) page
2. Import it through the in-game custom database/import flow
3. Restart TFM2 if the game asks you to reload custom data

### Manual AppData Path

The packaged release file is built for the game's import flow. If you manually edit AppData, back up the current file first and let the game create its own `custom_database.tfm2db` after importing.

```
C:\Users\<YourName>\AppData\Roaming\TeamSamoyed\TeamfightManager2\data\custom_database.tfm2db
```

## What's Included vs. What's Missing

### ✅ Working Features

| Feature | Status | Notes |
|---------|--------|-------|
| Team Names | ✅ Working | All 120 teams renamed |
| Team Logos | ✅ Working | Embedded custom logo assets, including all 120 verified custom logo blocks |
| Player Names | ✅ Working | Real IGNs for all players |
| Player Stats | ✅ Working | Multi-year Oracle's Elixir base with Games of Legends player/team margin adjustments |
| Position Skills | ✅ Working | Based on actual roles played |
| Player Ages | ✅ Working | Uses exact decoded age offsets |

### ⚠️ Known Limitations

| Feature | Status | Reason |
|---------|--------|--------|
| **Potential & Hidden Stats** | ⚠️ Partial | Some hidden fields could not be reliably modified |
| **Coach Names** | ⚠️ Partial | Some coaches may be missing or have default names |
| **Stadium Names** | ⚠️ Unchanged | Kept as game defaults |
| **Transfer History** | ❌ Not Included | Would require more research |

### ⚠️ Data Freshness

- **Data is from early 2026** - May not include latest transfers and roster changes
- New recruits from Summer split may be missing

### Data Quality Notes

- **Stats are approximations**: Real competitive performance is complex. We derived stats using formulas based on KDA, CS/min, vision score, damage share, and other metrics from Oracle's Elixir and Games of Legends.
- **League strength calibration**: LCK/LPL/LEC/LCS and other regions use explicit strength anchors, with Games of Legends team margins for within-region team separation.
- **Academy/Challenger players**: Division 2, minor, and regional players are capped lower than major-region starters to preserve real-world league gaps.
- **Role proficiency**: Based on games played per position, not necessarily player preference.

## Latest Release: v0.4.4.3

- Repacked the current verified logo database as a direct-import `.tfm2db` package (`kind=1`, gzip offset `25`).
- Preserved the validated v0.4.4.2 roster, age, contract, and balance data.
- Verified 120 team logo references and 120 embedded custom logo blocks after packaging.
- Fixed the SillySilly Gaming logo path through the import package using `custom:custom_team_logo/79`.
- Left stadium names unchanged after rolling back the experimental venue pass.
- Validated the packaged file with `tools/validate.go` and a decompressed-core comparison against the known-good logo candidate.
- Packaged database SHA256: `fde02f2821d47be645316f3e8335d388b068070e0351f5a4f3939b1292e28cb4`.

## For Modders: Create Your Own Roster

This project includes Go tools to unpack, edit, and repack tfm2db files.

### Prerequisites

- [Go 1.21+](https://go.dev/dl/) installed
- A base tfm2db file to start from

### Tool Usage

```bash
# 1. Unpack a tfm2db file to editable CSVs
go run unpack.go "input.tfm2db" "output_folder"

# 2. Edit the CSV files in output_folder:
#    - teams.csv    (team names, logos)
#    - athletes.csv (player names, stats)

# 3. Repack back into a tfm2db file
go run repack.go "output_folder" "modded.tfm2db"

# 4. Validate your modded file
go run validate.go "modded.tfm2db"
```

### CSV Field Reference

#### teams.csv
| Field | Editable | Notes |
|-------|----------|-------|
| team_name | ✅ Yes | Same length only |
| team_logo | ✅ Yes | `custom:custom_team_logo/N` or `X_Y` format |
| stadium_name | ✅ Yes | Same length only |
| manager_name | ✅ Yes | Same length only |

#### athletes.csv
| Field | Editable | Notes |
|-------|----------|-------|
| name | ✅ Yes | Player name |
| last_hit, skill_avoid, etc. | ✅ Yes | Stats (0-100 typical) |
| age | ✅ Yes | Writes only when `age_offset` is present |
| team_id | ✅ Yes | Contract field |
| face | ✅ Yes | Portrait index |

## Data Sources

- **Player/Team Data**: [Oracle's Elixir](https://oracleselixir.com/) - 2023-2026 competitive data
- **Secondary Balance Data**: [Games of Legends](https://gol.gg/) - public player and team aggregate tables
- **Team Logos**: Referenced from existing TFM2 custom logo format
- **Base File Format**: Reverse-engineered from TFM2 v0.4.1

## Technical Details

### TFM2DB File Format

```
Offset  Size  Description
0       4     Magic bytes "TFM2"
4       1     Kind byte (4 = game-saved custom database, 1 = packaged roster)
5       8     Timestamp (u64, milliseconds since epoch)
13      8     Gzip payload length (u64)
21      4     CRC32 of gzip payload
25/3484 ...   Gzip-compressed Rust bincode data, depending on package kind
```

### Athlete Data Structure

Each athlete stores ids before its name string, then visible stats and hidden fields after the name:
- Index 0: Version
- Index 1-20: Visible stats
- Index 21-31: Hidden stats
- Index 32-36: Contract fields
- Index 37: Face index
- Age is written only through the exact decoded `age_offset` exported by the unpacker.

## Disclaimer & Credits

### AI Assistance Disclosure

This project was developed with significant assistance from **GitHub Copilot (Claude Opus 4.5)**. The AI helped with:
- Reverse-engineering the tfm2db binary format
- Writing the Go unpacking/repacking tools
- Generating stat conversion formulas
- Debugging file corruption issues
- Writing this documentation

### Software Used

- **Python 3.12** - Initial tooling and data processing
- **Go 1.21** - Final clean tools for community use
- **Oracle's Elixir** - Competitive data source

### Legal

This mod is unofficial and not affiliated with Team Samoyed (TFM2 developers) or Riot Games (League of Legends). All team names, player names, and logos are property of their respective owners. This is a fan project for personal use.

### Known Issues

1. **Some players may have incorrect stats** - The formula is an approximation
2. **Academy players have less accurate data** - Fewer games to analyze
3. **Data from early 2026** - Some teams (FeelsStrongMan Los Ratones) no longer exist, latest transfers not included
4. **Some fields not editable** - Some hidden stats and unknown fields are intentionally preserved for stability

## Contributing

Found an issue? Have better data? Pull requests welcome!

1. Fork this repository
2. Edit the source CSVs or improve the tools
3. Test your changes in-game
4. Submit a PR with description of changes

## License

MIT License - See [LICENSE](LICENSE) for details.

---

*Last updated: May 2026*
