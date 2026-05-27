// TFM2DB Unpack Tool
// Extracts teams.csv and athletes.csv from a .tfm2db file for editing.
//
// Usage: go run unpack.go <input.tfm2db> [output_directory]
//
// This tool is part of the TFM2 Roster Mod project.
// https://github.com/eminyilmazz/tfm2-real-teams-and-rosters

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	TFM2Magic    = "TFM2"
	GzipMagic    = "\x1f\x8b\x08"
	HeaderSize   = 25
	MaxStrLength = 256
)

// Athlete stat field indices
const (
	AthleteStatVersionIndex   = 0
	AthleteStatStartIndex     = 1
	AthleteHiddenStartIndex   = 21 // 1 + 20 stat fields
	AthleteContractStartIndex = 32 // 21 + 11 hidden fields
	AthleteFaceIndex          = 37 // 32 + 5 contract fields
	AthleteDateTailIndex      = 38 // starts contract date strings or no-contract tail
	AthleteDetectU64Count     = AthleteContractStartIndex
	AthleteAgeDateStringCount = 4
	AthleteAgePostDatesBytes  = (2 * 8) + (4 * 8) + (8 * 8)
	AthleteAgeNoContractSkip  = 9
)

var (
	teamLogoRE = regexp.MustCompile(`^(?:custom:custom_team_logo/(\d+)|(\d+)_(\d+))$`)
	decoRE     = regexp.MustCompile(`^(?:#asset/|asset/|custom:|furniture_|wallpaper_|clean_|plain_|premium_|wide_window|basic_chair)`)
	dateRE     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

var athleteStatFields = []string{
	"last_hit", "skill_avoid", "skill_hit", "positioning", "control_speed",
	"concentration", "mental", "judgement", "order", "roaming",
	"aggressive", "ego", "top", "jungle", "mid", "bottom", "support",
	"like_champion", "dislike_champion", "language",
}

var athleteHiddenFields = []string{
	"potential", "stamina_recovery_max", "stamina_cost_per_set_min",
	"stamina_cost_per_set_max", "stress_sensitivity", "condition_baseline",
	"condition_amplitude", "condition_period", "condition_phase",
	"match_impact_sensitivity", "stamina_recovery_min",
}

var athleteContractFields = []string{
	"team_id", "end_date", "weekly_salary", "transfer_fee", "start_date",
}

// LPString represents a length-prefixed string found in the binary data
type LPString struct {
	Offset        int
	PayloadOffset int
	Length        int
	Text          string
}

// Team represents a team entry
type Team struct {
	Index          int
	NameOffset     int
	Name           string
	LogoOffset     int
	Logo           string
	StadiumOffset  int
	Stadium        string
	ManagerOffset  int
	Manager        string
	PreNameU64     [3]uint64
	RowStartGuess  int
	NextTeamOffset int
}

// Athlete represents an athlete entry
type Athlete struct {
	ID          uint64
	NameOffset  int
	Name        string
	Stats       map[string]uint64
	HiddenStats map[string]uint64
	Contract    map[string]uint64
	Face        uint64
	Age         uint32
	AgeOffset   int
	AfterName   int
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("TFM2DB Unpack Tool")
		fmt.Println("Usage: go run unpack.go <input.tfm2db> [output_directory]")
		fmt.Println("\nExtracts teams.csv and athletes.csv for editing.")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	outputDir := "."
	if len(os.Args) >= 3 {
		outputDir = os.Args[2]
	}

	// Read and decompress the file
	fmt.Printf("Reading %s...\n", inputPath)
	data, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Validate TFM2 header
	if !bytes.HasPrefix(data, []byte(TFM2Magic)) {
		fmt.Fprintf(os.Stderr, "Error: File does not have TFM2 magic header\n")
		os.Exit(1)
	}

	// Find gzip data
	gzipOffset := bytes.Index(data[4:], []byte(GzipMagic))
	if gzipOffset < 0 {
		fmt.Fprintf(os.Stderr, "Error: No gzip data found in file\n")
		os.Exit(1)
	}
	gzipOffset += 4

	// Decompress
	fmt.Println("Decompressing...")
	reader, err := gzip.NewReader(bytes.NewReader(data[gzipOffset:]))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating gzip reader: %v\n", err)
		os.Exit(1)
	}
	payload, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decompressing: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Decompressed %d bytes -> %d bytes\n", len(data)-gzipOffset, len(payload))

	// Scan for strings
	fmt.Println("Scanning for strings...")
	strings := scanLPStrings(payload)
	fmt.Printf("Found %d strings\n", len(strings))

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Find and export teams
	fmt.Println("Finding teams...")
	teams := findTeams(payload, strings)
	fmt.Printf("Found %d teams\n", len(teams))
	if err := exportTeams(teams, filepath.Join(outputDir, "teams.csv")); err != nil {
		fmt.Fprintf(os.Stderr, "Error exporting teams: %v\n", err)
		os.Exit(1)
	}

	// Find and export athletes
	fmt.Println("Finding athletes...")
	athletes := findAthletes(payload, strings)
	fmt.Printf("Found %d athletes\n", len(athletes))
	if err := exportAthletes(athletes, filepath.Join(outputDir, "athletes.csv")); err != nil {
		fmt.Fprintf(os.Stderr, "Error exporting athletes: %v\n", err)
		os.Exit(1)
	}

	// Save header for repacking
	headerPath := filepath.Join(outputDir, "header.bin")
	if err := os.WriteFile(headerPath, data[:gzipOffset], 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving header: %v\n", err)
		os.Exit(1)
	}

	// Save raw payload for repacking
	payloadPath := filepath.Join(outputDir, "payload.bin")
	if err := os.WriteFile(payloadPath, payload, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving payload: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nExported:")
	fmt.Printf("  - %s (teams)\n", filepath.Join(outputDir, "teams.csv"))
	fmt.Printf("  - %s (athletes)\n", filepath.Join(outputDir, "athletes.csv"))
	fmt.Printf("  - %s (original header)\n", headerPath)
	fmt.Printf("  - %s (raw payload for repack)\n", payloadPath)
	fmt.Println("\nEdit the CSV files, then run repack.go to create a new .tfm2db file.")
}

func scanLPStrings(data []byte) []LPString {
	var result []LPString
	dataLen := len(data)

	for offset := 0; offset < dataLen-8; offset++ {
		length := binary.LittleEndian.Uint64(data[offset:])
		if length < 1 || length > MaxStrLength {
			continue
		}
		end := offset + 8 + int(length)
		if end > dataLen {
			continue
		}
		raw := data[offset+8 : end]
		if !utf8.Valid(raw) {
			continue
		}
		text := string(raw)
		if !isPlausibleText(text) {
			continue
		}
		result = append(result, LPString{
			Offset:        offset,
			PayloadOffset: offset + 8,
			Length:        int(length),
			Text:          text,
		})
	}
	return result
}

func isPlausibleText(text string) bool {
	if strings.Contains(text, "\ufffd") {
		return false
	}
	if strings.TrimSpace(text) == "" {
		return false
	}
	for _, r := range text {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}

func hasLetter(text string) bool {
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

func isMeaningfulNonAsset(text string) bool {
	trimmed := strings.TrimSpace(text)
	return len(trimmed) >= 3 && hasLetter(text) && !decoRE.MatchString(text)
}

func parseTeamLogoIndex(text string) (int, bool) {
	match := teamLogoRE.FindStringSubmatch(text)
	if match == nil {
		return 0, false
	}
	if match[1] != "" {
		idx, _ := strconv.Atoi(match[1])
		return idx, true
	}
	x, _ := strconv.Atoi(match[2])
	y, _ := strconv.Atoi(match[3])
	return x + y*10, true
}

func readU64(data []byte, offset int) (uint64, bool) {
	if offset < 0 || offset+8 > len(data) {
		return 0, false
	}
	return binary.LittleEndian.Uint64(data[offset:]), true
}

func findTeams(data []byte, strings []LPString) []Team {
	// Find all logo entries
	type logoEntry struct {
		stringIndex int
		logoIndex   int
		str         LPString
	}
	var logoEntries []logoEntry
	for i, s := range strings {
		if idx, ok := parseTeamLogoIndex(s.Text); ok {
			logoEntries = append(logoEntries, logoEntry{i, idx, s})
		}
	}

	var teams []Team
	for entryIdx, entry := range logoEntries {
		if entry.stringIndex == 0 {
			continue
		}
		name := strings[entry.stringIndex-1]
		if entry.str.PayloadOffset-name.PayloadOffset > 256 {
			continue
		}

		// Find next team boundary
		var nextNameOffset int = len(data)
		if entryIdx+1 < len(logoEntries) {
			nextIdx := logoEntries[entryIdx+1].stringIndex
			if nextIdx > 0 {
				nextNameOffset = strings[nextIdx-1].Offset
			}
		}

		// Find stadium and manager
		var stadium, manager *LPString
		for i := entry.stringIndex + 1; i < len(strings) && strings[i].Offset < nextNameOffset; i++ {
			if isMeaningfulNonAsset(strings[i].Text) {
				if stadium == nil {
					s := strings[i]
					stadium = &s
				} else if manager == nil {
					s := strings[i]
					manager = &s
				} else {
					break
				}
			}
		}

		preNameOffset := name.Offset - 24
		team := Team{
			Index:          entry.logoIndex,
			NameOffset:     name.Offset,
			Name:           name.Text,
			LogoOffset:     entry.str.Offset,
			Logo:           entry.str.Text,
			NextTeamOffset: nextNameOffset,
		}
		if preNameOffset >= 0 {
			team.RowStartGuess = preNameOffset
			team.PreNameU64[0], _ = readU64(data, preNameOffset)
			team.PreNameU64[1], _ = readU64(data, preNameOffset+8)
			team.PreNameU64[2], _ = readU64(data, preNameOffset+16)
		}
		if stadium != nil {
			team.StadiumOffset = stadium.Offset
			team.Stadium = stadium.Text
		}
		if manager != nil {
			team.ManagerOffset = manager.Offset
			team.Manager = manager.Text
		}
		teams = append(teams, team)
	}

	sort.Slice(teams, func(i, j int) bool {
		return teams[i].Index < teams[j].Index
	})
	return teams
}

func isPlausibleAthleteName(text string) bool {
	l := len(text)
	if l < 2 || l > 32 {
		return false
	}
	if !hasLetter(text) || decoRE.MatchString(text) {
		return false
	}
	if strings.ContainsAny(text, "#?/:\\") {
		return false
	}
	if dateRE.MatchString(text) {
		return false
	}
	return true
}

func smallU64Run(data []byte, offset, count int, maximum uint64) bool {
	for i := 0; i < count; i++ {
		v, ok := readU64(data, offset+i*8)
		if !ok || v > maximum {
			return false
		}
	}
	return true
}

func decodeAgeAtOffset(data []byte, offset int) (uint32, bool) {
	if offset <= 0 || offset+4 > len(data) {
		return 0, false
	}
	age := binary.LittleEndian.Uint32(data[offset:])
	if age > 120 {
		return 0, false
	}
	return age, true
}

func findAgeOffset(data []byte, afterName int) (int, uint32, bool) {
	dateTailOffset := afterName + AthleteDateTailIndex*8
	first, ok := readU64(data, dateTailOffset)
	if !ok {
		return 0, 0, false
	}

	if first == 0 {
		ageOffset := dateTailOffset + AthleteAgeNoContractSkip*8
		age, ok := decodeAgeAtOffset(data, ageOffset)
		return ageOffset, age, ok
	}

	offset := dateTailOffset
	for i := 0; i < AthleteAgeDateStringCount; i++ {
		length, ok := readU64(data, offset)
		if !ok || length > 64 {
			return 0, 0, false
		}
		end := offset + 8 + int(length)
		if end > len(data) || !utf8.Valid(data[offset+8:end]) {
			return 0, 0, false
		}
		offset = end
	}

	ageOffset := offset + AthleteAgePostDatesBytes
	age, ok := decodeAgeAtOffset(data, ageOffset)
	return ageOffset, age, ok
}

func findAthletes(data []byte, strings []LPString) []Athlete {
	var athletes []Athlete
	seenOffsets := make(map[int]bool)

	for _, s := range strings {
		if !isPlausibleAthleteName(s.Text) {
			continue
		}

		idA, okA := readU64(data, s.Offset-24)
		version, okB := readU64(data, s.Offset-16)
		idC, okC := readU64(data, s.Offset-8)
		if !okA || !okB || !okC || idA != idC || idA > 100000 || version > 10000 {
			continue
		}

		afterName := s.Offset + 8 + s.Length
		if !smallU64Run(data, afterName, AthleteDetectU64Count, 1000) {
			continue
		}
		if seenOffsets[s.Offset] {
			continue
		}
		seenOffsets[s.Offset] = true

		ageOffset, age, _ := findAgeOffset(data, afterName)

		athlete := Athlete{
			ID:          idA,
			NameOffset:  s.Offset,
			Name:        s.Text,
			AfterName:   afterName,
			Age:         age,
			AgeOffset:   ageOffset,
			Stats:       make(map[string]uint64),
			HiddenStats: make(map[string]uint64),
			Contract:    make(map[string]uint64),
		}

		// Read stats
		for i, field := range athleteStatFields {
			v, _ := readU64(data, afterName+(AthleteStatStartIndex+i)*8)
			athlete.Stats[field] = v
		}

		// Read hidden stats
		for i, field := range athleteHiddenFields {
			v, _ := readU64(data, afterName+(AthleteHiddenStartIndex+i)*8)
			athlete.HiddenStats[field] = v
		}

		// Read contract fields
		for i, field := range athleteContractFields {
			v, _ := readU64(data, afterName+(AthleteContractStartIndex+i)*8)
			athlete.Contract[field] = v
		}

		// Read face
		athlete.Face, _ = readU64(data, afterName+AthleteFaceIndex*8)

		athletes = append(athletes, athlete)
	}

	return athletes
}

func exportTeams(teams []Team, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write BOM for Excel compatibility
	file.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Header
	header := []string{
		"team_index", "team_name_offset", "team_name", "team_logo_offset", "team_logo",
		"stadium_name_offset", "stadium_name", "manager_name_offset", "manager_name",
	}
	writer.Write(header)

	// Data
	for _, t := range teams {
		row := []string{
			strconv.Itoa(t.Index),
			strconv.Itoa(t.NameOffset),
			t.Name,
			strconv.Itoa(t.LogoOffset),
			t.Logo,
			intOrEmpty(t.StadiumOffset),
			t.Stadium,
			intOrEmpty(t.ManagerOffset),
			t.Manager,
		}
		writer.Write(row)
	}
	return nil
}

func exportAthletes(athletes []Athlete, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write BOM for Excel compatibility
	file.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Build header
	header := []string{"name_offset", "name", "age", "age_offset"}
	header = append(header, athleteStatFields...)
	header = append(header, athleteHiddenFields...)
	header = append(header, athleteContractFields...)
	header = append(header, "face")
	writer.Write(header)

	// Data
	for _, a := range athletes {
		row := []string{
			strconv.Itoa(a.NameOffset),
			a.Name,
			uint32OrEmpty(a.AgeOffset, a.Age),
			intOrEmpty(a.AgeOffset),
		}
		for _, field := range athleteStatFields {
			row = append(row, strconv.FormatUint(a.Stats[field], 10))
		}
		for _, field := range athleteHiddenFields {
			row = append(row, strconv.FormatUint(a.HiddenStats[field], 10))
		}
		for _, field := range athleteContractFields {
			row = append(row, strconv.FormatUint(a.Contract[field], 10))
		}
		row = append(row, strconv.FormatUint(a.Face, 10))
		writer.Write(row)
	}
	return nil
}

func intOrEmpty(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func uint32OrEmpty(offset int, n uint32) string {
	if offset == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(n), 10)
}
