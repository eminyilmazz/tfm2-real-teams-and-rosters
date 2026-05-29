// TFM2DB Validate Tool
// Validates a .tfm2db file structure and reports any issues.
//
// Usage:
//   go run validate.go [flags] <input.tfm2db>
//
// This tool is part of the TFM2 Roster Mod project.
// https://github.com/eminyilmazz/tfm2-real-teams-and-rosters

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	TFM2Magic           = "TFM2"
	GzipMagic           = "\x1f\x8b\x08"
	HeaderSize          = 25
	ImportGzipOffset    = 25
	AppDataGzipOffset   = 3484
	MaxStrLength        = 256
	WideLogoWarnRatio   = 4.0
	NarrowLogoWarnRatio = 0.25
)

var (
	pngMagic   = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	teamLogoRE = regexp.MustCompile(`^(?:custom:custom_team_logo/(\d+)|(\d+)_(\d+))$`)
	decoRE     = regexp.MustCompile(`^(?:#asset/|asset/|custom:|furniture_|wallpaper_|clean_|plain_|premium_|wide_window|basic_chair)`)
)

type options struct {
	strict                   bool
	expectedKind             int
	expectedTeamCount        int
	expectedCustomLogoBlocks int
	minAthletes              int
	maxDefaultLogoRefs       int
	allowDefaultLogoTeams    stringList
	requireCustomLogoTeams   stringList
}

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("empty team name")
	}
	*s = append(*s, value)
	return nil
}

type validationResult struct {
	errors   []string
	warnings []string
	info     []string
}

type lpString struct {
	offset        int
	payloadOffset int
	length        int
	text          string
}

type team struct {
	index          int
	nameOffset     int
	name           string
	logoOffset     int
	logo           string
	stadiumOffset  int
	stadium        string
	managerOffset  int
	manager        string
	preNameU64     [3]uint64
	rowStartGuess  int
	nextTeamOffset int
}

type teamLogoRef struct {
	raw      string
	custom   bool
	id       int
	defaultX int
	defaultY int
}

type customLogoBlock struct {
	logoID       int
	idOffset     int
	lengthOffset int
	start        int
	end          int
	length       int
	width        int
	height       int
}

func (v *validationResult) addError(msg string) {
	v.errors = append(v.errors, msg)
}

func (v *validationResult) addWarning(msg string) {
	v.warnings = append(v.warnings, msg)
}

func (v *validationResult) addInfo(msg string) {
	v.info = append(v.info, msg)
}

func main() {
	opts := options{
		expectedKind:       -1,
		minAthletes:        500,
		maxDefaultLogoRefs: -1,
	}
	flag.BoolVar(&opts.strict, "strict", false, "treat header checksum/length mismatches and expected-count failures as errors")
	flag.IntVar(&opts.expectedKind, "expected-kind", opts.expectedKind, "expected TFM2 kind byte; use 1 for import packages, 4 for AppData saves")
	flag.IntVar(&opts.expectedTeamCount, "expected-team-count", 0, "expected parsed team count; 0 disables exact count")
	flag.IntVar(&opts.expectedCustomLogoBlocks, "expected-custom-logo-blocks", 0, "expected embedded custom PNG block count; 0 disables exact count")
	flag.IntVar(&opts.minAthletes, "min-athletes", opts.minAthletes, "minimum athlete-like row count")
	flag.IntVar(&opts.maxDefaultLogoRefs, "max-default-logo-refs", opts.maxDefaultLogoRefs, "maximum team rows allowed to use default X_Y logos; -1 disables")
	flag.Var(&opts.allowDefaultLogoTeams, "allow-default-logo-team", "team name allowed to use a default X_Y logo ref; repeatable")
	flag.Var(&opts.requireCustomLogoTeams, "require-custom-logo-team", "team name that must use a custom:custom_team_logo/N ref; repeatable")
	flag.Usage = func() {
		fmt.Println("TFM2DB Validate Tool")
		fmt.Println("Usage: go run validate.go [flags] <input.tfm2db>")
		fmt.Println("\nValidates a .tfm2db file and reports structural, roster-table, and logo issues.")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if opts.strict {
		if opts.expectedKind < 0 {
			opts.expectedKind = 1
		}
		if opts.expectedTeamCount == 0 {
			opts.expectedTeamCount = 120
		}
		if opts.expectedCustomLogoBlocks == 0 {
			opts.expectedCustomLogoBlocks = 120
		}
		if opts.minAthletes < 1100 {
			opts.minAthletes = 1100
		}
	}

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputPath := flag.Arg(0)
	result := &validationResult{}

	fmt.Printf("Validating %s...\n\n", inputPath)

	data, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	result.addInfo(fmt.Sprintf("File size: %d bytes", len(data)))

	kind := validateHeader(data, opts, result)
	if len(result.errors) > 0 {
		printResults(result)
		os.Exit(1)
	}

	gzipOffset := bytes.Index(data[4:], []byte(GzipMagic))
	if gzipOffset < 0 {
		result.addError("No gzip data found in file")
		printResults(result)
		os.Exit(1)
	}
	gzipOffset += 4
	result.addInfo(fmt.Sprintf("Gzip offset: %d", gzipOffset))
	validateGzipOffset(kind, gzipOffset, opts, result)

	if gzipOffset >= HeaderSize {
		validateHeaderFields(data, gzipOffset, opts, result)
	}

	payload := validateDecompression(data[gzipOffset:], result)
	if payload == nil {
		printResults(result)
		os.Exit(1)
	}
	result.addInfo(fmt.Sprintf("Decompressed size: %d bytes", len(payload)))

	validateContent(payload, opts, result)
	printResults(result)

	if len(result.errors) > 0 {
		os.Exit(1)
	}
}

func validateHeader(data []byte, opts options, result *validationResult) byte {
	if len(data) < 4 {
		result.addError("File too small - missing TFM2 header")
		return 0
	}

	magic := string(data[:4])
	if magic != TFM2Magic {
		result.addError(fmt.Sprintf("Invalid magic: expected 'TFM2', got '%s'", magic))
		return 0
	}
	result.addInfo("Magic: TFM2")

	if len(data) < HeaderSize {
		result.addWarning(fmt.Sprintf("Header smaller than expected: %d < %d bytes", len(data), HeaderSize))
		return 0
	}

	kind := data[4]
	result.addInfo(fmt.Sprintf("Kind byte: %d", kind))
	if opts.expectedKind >= 0 && int(kind) != opts.expectedKind {
		result.addError(fmt.Sprintf("Kind byte mismatch: expected %d, got %d", opts.expectedKind, kind))
	}
	return kind
}

func validateGzipOffset(kind byte, gzipOffset int, opts options, result *validationResult) {
	expectedOffset := -1
	if opts.expectedKind == 1 || (opts.expectedKind < 0 && kind == 1) {
		expectedOffset = ImportGzipOffset
	} else if opts.expectedKind == 4 || (opts.expectedKind < 0 && kind == 4) {
		expectedOffset = AppDataGzipOffset
	}
	if expectedOffset < 0 || gzipOffset == expectedOffset {
		return
	}

	message := fmt.Sprintf("Gzip offset mismatch for kind %d: expected %d, got %d", kind, expectedOffset, gzipOffset)
	if opts.strict || opts.expectedKind >= 0 {
		result.addError(message)
	} else {
		result.addWarning(message)
	}
}

func validateHeaderFields(data []byte, gzipOffset int, opts options, result *validationResult) {
	timestamp := binary.LittleEndian.Uint64(data[5:13])
	result.addInfo(fmt.Sprintf("Timestamp: %d", timestamp))

	storedGzLen := binary.LittleEndian.Uint64(data[13:21])
	actualGzLen := len(data) - gzipOffset
	result.addInfo(fmt.Sprintf("Stored gzip length: %d", storedGzLen))
	result.addInfo(fmt.Sprintf("Actual gzip length: %d", actualGzLen))
	if storedGzLen != uint64(actualGzLen) {
		message := fmt.Sprintf("Gzip length mismatch: stored %d != actual %d", storedGzLen, actualGzLen)
		if opts.strict {
			result.addError(message)
		} else {
			result.addWarning(message)
		}
	}

	storedCRC := binary.LittleEndian.Uint32(data[21:25])
	actualCRC := crc32.ChecksumIEEE(data[gzipOffset:])
	result.addInfo(fmt.Sprintf("Stored CRC32: %d", storedCRC))
	result.addInfo(fmt.Sprintf("Actual CRC32: %d", actualCRC))
	if storedCRC != actualCRC {
		message := fmt.Sprintf("CRC32 mismatch: stored %d != actual %d", storedCRC, actualCRC)
		if opts.strict {
			result.addError(message)
		} else {
			result.addWarning(message)
		}
	}
}

func validateDecompression(gzipData []byte, result *validationResult) []byte {
	reader, err := gzip.NewReader(bytes.NewReader(gzipData))
	if err != nil {
		result.addError(fmt.Sprintf("Failed to create gzip reader: %v", err))
		return nil
	}
	defer reader.Close()

	payload, err := io.ReadAll(reader)
	if err != nil {
		result.addError(fmt.Sprintf("Failed to decompress: %v", err))
		return nil
	}

	result.addInfo("Decompression: OK")
	return payload
}

func validateContent(payload []byte, opts options, result *validationResult) {
	lpStrings := scanLPStrings(payload)
	result.addInfo(fmt.Sprintf("Strings found: %d", len(lpStrings)))

	teams := findTeams(payload, lpStrings)
	result.addInfo(fmt.Sprintf("Teams parsed: %d", len(teams)))
	if opts.expectedTeamCount > 0 && len(teams) != opts.expectedTeamCount {
		result.addError(fmt.Sprintf("Team count mismatch: expected %d, got %d", opts.expectedTeamCount, len(teams)))
	} else if len(teams) == 0 {
		result.addWarning("No teams found - file may be corrupted or empty")
	} else if len(teams) < 100 {
		result.addWarning(fmt.Sprintf("Only %d teams found (expected ~120)", len(teams)))
	} else if len(teams) > 200 {
		result.addWarning(fmt.Sprintf("Unusually many teams found: %d", len(teams)))
	}

	logoBlocks := scanCustomLogoBlocks(payload, result)
	result.addInfo(fmt.Sprintf("Embedded custom logo PNG blocks: %d", len(logoBlocks)))
	if opts.expectedCustomLogoBlocks > 0 && len(logoBlocks) != opts.expectedCustomLogoBlocks {
		result.addError(fmt.Sprintf("Custom logo block count mismatch: expected %d, got %d", opts.expectedCustomLogoBlocks, len(logoBlocks)))
	}
	validateTeamLogos(teams, logoBlocks, opts, result)

	athleteCount := countAthletes(payload)
	result.addInfo(fmt.Sprintf("Athlete candidates: %d", athleteCount))
	if athleteCount == 0 {
		result.addWarning("No athletes found - file may be corrupted")
	} else if athleteCount < opts.minAthletes {
		message := fmt.Sprintf("Only %d athletes found (minimum %d)", athleteCount, opts.minAthletes)
		if opts.strict {
			result.addError(message)
		} else {
			result.addWarning(message)
		}
	}

	checkCorruption(payload, result)
}

func scanLPStrings(data []byte) []lpString {
	var result []lpString
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
		result = append(result, lpString{
			offset:        offset,
			payloadOffset: offset + 8,
			length:        int(length),
			text:          text,
		})
	}
	return result
}

func isPlausibleText(text string) bool {
	if strings.Contains(text, "\ufffd") || strings.TrimSpace(text) == "" {
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
	ref, ok := parseTeamLogoRef(text)
	if !ok {
		return 0, false
	}
	return ref.id, true
}

func parseTeamLogoRef(text string) (teamLogoRef, bool) {
	match := teamLogoRE.FindStringSubmatch(text)
	if match == nil {
		return teamLogoRef{}, false
	}
	if match[1] != "" {
		idx, _ := strconv.Atoi(match[1])
		return teamLogoRef{raw: text, custom: true, id: idx}, true
	}
	x, _ := strconv.Atoi(match[2])
	y, _ := strconv.Atoi(match[3])
	return teamLogoRef{raw: text, custom: false, id: x + y*10, defaultX: x, defaultY: y}, true
}

func readU64(data []byte, offset int) (uint64, bool) {
	if offset < 0 || offset+8 > len(data) {
		return 0, false
	}
	return binary.LittleEndian.Uint64(data[offset:]), true
}

func findTeams(data []byte, lpStrings []lpString) []team {
	type logoEntry struct {
		stringIndex int
		logoIndex   int
		str         lpString
	}
	var logoEntries []logoEntry
	for i, s := range lpStrings {
		if idx, ok := parseTeamLogoIndex(s.text); ok {
			logoEntries = append(logoEntries, logoEntry{i, idx, s})
		}
	}

	var teams []team
	for entryIdx, entry := range logoEntries {
		if entry.stringIndex == 0 {
			continue
		}
		name := lpStrings[entry.stringIndex-1]
		if entry.str.payloadOffset-name.payloadOffset > 256 {
			continue
		}

		nextNameOffset := len(data)
		if entryIdx+1 < len(logoEntries) {
			nextIdx := logoEntries[entryIdx+1].stringIndex
			if nextIdx > 0 {
				nextNameOffset = lpStrings[nextIdx-1].offset
			}
		}

		var stadium, manager *lpString
		for i := entry.stringIndex + 1; i < len(lpStrings) && lpStrings[i].offset < nextNameOffset; i++ {
			if isMeaningfulNonAsset(lpStrings[i].text) {
				if stadium == nil {
					s := lpStrings[i]
					stadium = &s
				} else if manager == nil {
					s := lpStrings[i]
					manager = &s
				} else {
					break
				}
			}
		}

		preNameOffset := name.offset - 24
		team := team{
			index:          entry.logoIndex,
			nameOffset:     name.offset,
			name:           name.text,
			logoOffset:     entry.str.offset,
			logo:           entry.str.text,
			nextTeamOffset: nextNameOffset,
		}
		if preNameOffset >= 0 {
			team.rowStartGuess = preNameOffset
			team.preNameU64[0], _ = readU64(data, preNameOffset)
			team.preNameU64[1], _ = readU64(data, preNameOffset+8)
			team.preNameU64[2], _ = readU64(data, preNameOffset+16)
		}
		if stadium != nil {
			team.stadiumOffset = stadium.offset
			team.stadium = stadium.text
		}
		if manager != nil {
			team.managerOffset = manager.offset
			team.manager = manager.text
		}
		teams = append(teams, team)
	}

	sort.Slice(teams, func(i, j int) bool {
		return teams[i].index < teams[j].index
	})
	return teams
}

func scanCustomLogoBlocks(data []byte, result *validationResult) map[int]customLogoBlock {
	blocks := make(map[int]customLogoBlock)
	searchFrom := 0
	for searchFrom < len(data) {
		relativeStart := bytes.Index(data[searchFrom:], pngMagic)
		if relativeStart < 0 {
			break
		}
		start := searchFrom + relativeStart
		end, width, height, err := pngPayloadEnd(data, start)
		if err != nil {
			result.addError(fmt.Sprintf("Invalid PNG payload at offset %d: %v", start, err))
			searchFrom = start + len(pngMagic)
			continue
		}

		if start >= 16 {
			logoID := binary.LittleEndian.Uint64(data[start-16:])
			length := binary.LittleEndian.Uint64(data[start-8:])
			if length == uint64(end-start) {
				id := int(logoID)
				if existing, ok := blocks[id]; ok {
					result.addError(fmt.Sprintf("Duplicate custom logo id %d at offsets %d and %d", id, existing.start, start))
				} else {
					blocks[id] = customLogoBlock{
						logoID:       id,
						idOffset:     start - 16,
						lengthOffset: start - 8,
						start:        start,
						end:          end,
						length:       end - start,
						width:        width,
						height:       height,
					}
				}
			}
		}
		searchFrom = end
	}
	return blocks
}

func pngPayloadEnd(data []byte, start int) (int, int, int, error) {
	if start < 0 || start+len(pngMagic) > len(data) || !bytes.Equal(data[start:start+len(pngMagic)], pngMagic) {
		return 0, 0, 0, fmt.Errorf("missing PNG signature")
	}
	if start+33 > len(data) {
		return 0, 0, 0, fmt.Errorf("truncated before IHDR")
	}
	ihdrLength := binary.BigEndian.Uint32(data[start+8:])
	ihdrType := string(data[start+12 : start+16])
	if ihdrLength != 13 || ihdrType != "IHDR" {
		return 0, 0, 0, fmt.Errorf("first chunk is %q length %d, expected IHDR length 13", ihdrType, ihdrLength)
	}
	width := int(binary.BigEndian.Uint32(data[start+16:]))
	height := int(binary.BigEndian.Uint32(data[start+20:]))
	if width <= 0 || height <= 0 {
		return 0, 0, 0, fmt.Errorf("invalid dimensions %dx%d", width, height)
	}

	offset := start + len(pngMagic)
	for offset+12 <= len(data) {
		chunkLength := int(binary.BigEndian.Uint32(data[offset:]))
		chunkType := string(data[offset+4 : offset+8])
		nextOffset := offset + 12 + chunkLength
		if nextOffset > len(data) {
			return 0, 0, 0, fmt.Errorf("chunk %q at offset %d extends past available data", chunkType, offset)
		}
		if chunkType == "IEND" {
			return nextOffset, width, height, nil
		}
		offset = nextOffset
	}
	return 0, 0, 0, fmt.Errorf("missing complete IEND chunk")
}

func validateTeamLogos(teams []team, logoBlocks map[int]customLogoBlock, opts options, result *validationResult) {
	customRefsByID := make(map[int][]string)
	teamByLowerName := make(map[string]team)
	allowedDefaultTeams := make(map[string]bool)
	var defaultRefs []string
	for _, teamName := range opts.allowDefaultLogoTeams {
		allowedDefaultTeams[strings.ToLower(teamName)] = true
	}

	for _, team := range teams {
		teamByLowerName[strings.ToLower(team.name)] = team
		ref, ok := parseTeamLogoRef(team.logo)
		if !ok {
			result.addError(fmt.Sprintf("Team %q has invalid logo ref %q", team.name, team.logo))
			continue
		}
		if ref.custom {
			customRefsByID[ref.id] = append(customRefsByID[ref.id], team.name)
			if _, ok := logoBlocks[ref.id]; !ok {
				result.addError(fmt.Sprintf("Team %q references missing custom logo id %d", team.name, ref.id))
			}
		} else {
			defaultRefs = append(defaultRefs, fmt.Sprintf("%s=%s", team.name, team.logo))
			if len(allowedDefaultTeams) > 0 && !allowedDefaultTeams[strings.ToLower(team.name)] {
				result.addError(fmt.Sprintf("Team %q uses default logo ref %q but is not in the default-logo allowlist", team.name, team.logo))
			}
		}
	}

	result.addInfo(fmt.Sprintf("Teams using custom logo refs: %d", len(teams)-len(defaultRefs)))
	result.addInfo(fmt.Sprintf("Teams using default logo refs: %d", len(defaultRefs)))
	if len(defaultRefs) > 0 {
		result.addWarning(fmt.Sprintf("Default logo refs still present: %s", summarizeStrings(defaultRefs, 8)))
	}
	if opts.maxDefaultLogoRefs >= 0 && len(defaultRefs) > opts.maxDefaultLogoRefs {
		result.addError(fmt.Sprintf("Default logo ref count %d exceeds allowed maximum %d", len(defaultRefs), opts.maxDefaultLogoRefs))
	}

	for _, teamName := range opts.requireCustomLogoTeams {
		team, ok := teamByLowerName[strings.ToLower(teamName)]
		if !ok {
			result.addError(fmt.Sprintf("Required custom-logo team %q was not found", teamName))
			continue
		}
		ref, ok := parseTeamLogoRef(team.logo)
		if !ok || !ref.custom {
			result.addError(fmt.Sprintf("Team %q must use a custom logo ref, got %q", team.name, team.logo))
		}
	}

	var sharedRefs []string
	for id, teamNames := range customRefsByID {
		if len(teamNames) > 1 {
			sort.Strings(teamNames)
			sharedRefs = append(sharedRefs, fmt.Sprintf("%d=%s", id, strings.Join(teamNames, "/")))
		}
	}
	sort.Strings(sharedRefs)
	if len(sharedRefs) > 0 {
		result.addInfo(fmt.Sprintf("Shared custom logo refs: %s", summarizeStrings(sharedRefs, 8)))
	}

	var unusualAspect []string
	for id, block := range logoBlocks {
		ratio := float64(block.width) / float64(block.height)
		if ratio > WideLogoWarnRatio || ratio < NarrowLogoWarnRatio {
			unusualAspect = append(unusualAspect, fmt.Sprintf("%d=%dx%d", id, block.width, block.height))
		}
	}
	sort.Strings(unusualAspect)
	if len(unusualAspect) > 0 {
		result.addWarning(fmt.Sprintf("Very wide/tall embedded logo payloads: %s", summarizeStrings(unusualAspect, 12)))
	}

	if len(logoBlocks) > 0 {
		ids := make([]int, 0, len(logoBlocks))
		for id := range logoBlocks {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		result.addInfo(fmt.Sprintf("Custom logo id range: %d..%d", ids[0], ids[len(ids)-1]))
	}
}

func summarizeStrings(values []string, limit int) string {
	if len(values) <= limit {
		return strings.Join(values, ", ")
	}
	return strings.Join(values[:limit], ", ") + fmt.Sprintf(", ... (%d total)", len(values))
}

func countAthletes(data []byte) int {
	count := 0
	dataLen := len(data)
	for offset := 0; offset < dataLen-8; offset++ {
		length := binary.LittleEndian.Uint64(data[offset:])
		if length < 2 || length > 32 {
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
		if !hasLetter(text) || decoRE.MatchString(text) {
			continue
		}

		preOffset := offset - 39*8
		if preOffset < 0 {
			continue
		}

		looksLikeStats := true
		for i := 0; i < 32; i++ {
			v := binary.LittleEndian.Uint64(data[preOffset+i*8:])
			if v > 1000 {
				looksLikeStats = false
				break
			}
		}
		if looksLikeStats {
			count++
		}
	}
	return count
}

func checkCorruption(payload []byte, result *validationResult) {
	zeroRun := 0
	maxZeroRun := 0
	for _, b := range payload {
		if b == 0 {
			zeroRun++
			if zeroRun > maxZeroRun {
				maxZeroRun = zeroRun
			}
		} else {
			zeroRun = 0
		}
	}
	if maxZeroRun > 1000 {
		result.addWarning(fmt.Sprintf("Large run of zeros detected: %d bytes (possible corruption)", maxZeroRun))
	}

	nullCount := 0
	for _, b := range payload {
		if b == 0 {
			nullCount++
		}
	}
	nullPercent := float64(nullCount) / float64(len(payload)) * 100
	result.addInfo(fmt.Sprintf("Null bytes: %.1f%%", nullPercent))
	if nullPercent > 50 {
		result.addWarning("High percentage of null bytes - file may be corrupted")
	}
}

func printResults(result *validationResult) {
	fmt.Println("=== VALIDATION RESULTS ===")
	fmt.Println()

	if len(result.info) > 0 {
		fmt.Println("INFO:")
		for _, msg := range result.info {
			fmt.Printf("  - %s\n", msg)
		}
		fmt.Println()
	}

	if len(result.warnings) > 0 {
		fmt.Println("WARNINGS:")
		for _, msg := range result.warnings {
			fmt.Printf("  - %s\n", msg)
		}
		fmt.Println()
	}

	if len(result.errors) > 0 {
		fmt.Println("ERRORS:")
		for _, msg := range result.errors {
			fmt.Printf("  - %s\n", msg)
		}
		fmt.Println()
	}

	if len(result.errors) == 0 {
		fmt.Println("Validation PASSED")
	} else {
		fmt.Println("Validation FAILED")
	}
}
