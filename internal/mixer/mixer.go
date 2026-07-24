// Package mixer provides word combination generation with multiple casing
// modes, number appending, special character combinations, and configurable
// combination depth. It supports permutations and subsets of input words.
package mixer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"wordlist-generator/internal/ui"
)

// specialChars defines the available special characters that can be appended
const specialChars = "!?$.@#*&+="

// Config holds all parameters for a mixer generation run.
type Config struct {
	Words      []string    // input words to combine
	MaxCap     int          // max character length per output word (0 = no cap)
	CasingIdxs []int        // which casing modes to apply (0=Pascal, 1=camel, 2=UPPER, 3=lower)
	AppendNums bool        // whether to append numbers to words
	Numbers    []int       // the numbers to append
	AppendSpec bool        // whether to append special character combinations
	SpecCombos []string    // pre-computed special character combinations
	MaxSpecLen int         // max number of special chars to combine
	Sizes      []int       // combination sizes (1=single, 2=two-word, 3=three-word, etc.)
	Combos     [][]string  // pre-computed word combinations
	MaxBytes   uint64      // max total output size in bytes (0 = no cap)
}

// toPascal capitalizes the first letter and preserves the rest of the word.
// Uses []rune for unicode safety (handles multi-byte characters like é, ñ).
func toPascal(w string) string {
	if w == "" {
		return ""
	}
	r := []rune(w)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// toCamelFirst lowercases the first letter and preserves the rest.
// Used for the first word in camelCase mode.
func toCamelFirst(w string) string {
	if w == "" {
		return ""
	}
	r := []rune(w)
	return strings.ToLower(string(r[0])) + string(r[1:])
}

// ApplyCasing applies a casing mode to a slice of words.
// casing: 0=PascalCase, 1=camelCase, 2=UPPER, 3=lower
func ApplyCasing(words []string, casing int) []string {
	out := make([]string, len(words))
	for i, w := range words {
		switch casing {
		case 0:
			out[i] = toPascal(w)
		case 1:
			if i == 0 {
				out[i] = toCamelFirst(w)
			} else {
				out[i] = toPascal(w)
			}
		case 2:
			out[i] = strings.ToUpper(w)
		case 3:
			out[i] = strings.ToLower(w)
		}
	}
	return out
}

// CasingName returns the display name for a casing mode index.
func CasingName(c int) string {
	switch c {
	case 0:
		return "PascalCase"
	case 1:
		return "camelCase"
	case 2:
		return "UPPER"
	case 3:
		return "lower"
	}
	return "?"
}

// GeneratePermutations returns all permutations of the input items using
// Heap's algorithm. Returns n! permutations for n items.
func GeneratePermutations(items []string) [][]string {
	n := len(items)
	if n == 0 {
		return nil
	}
	if n == 1 {
		return [][]string{{items[0]}}
	}
	arr := make([]string, n)
	copy(arr, items)
	var result [][]string
	var gen func(k int)
	gen = func(k int) {
		if k == 1 {
			perm := make([]string, n)
			copy(perm, arr)
			result = append(result, perm)
			return
		}
		gen(k - 1)
		for i := 0; i < k-1; i++ {
			if k%2 == 0 {
				arr[i], arr[k-1] = arr[k-1], arr[i]
			} else {
				arr[0], arr[k-1] = arr[k-1], arr[0]
			}
			gen(k - 1)
		}
	}
	gen(n)
	return result
}

// Subsets returns all subsets of size k from n elements (as index slices).
// Each subset is a slice of indices into the original n elements.
// Returns nil if k > n or k < 1.
func Subsets(n, k int) [][]int {
	if k > n || k < 1 {
		return nil
	}
	var result [][]int
	var gen func(start int, current []int)
	gen = func(start int, current []int) {
		if len(current) == k {
			cp := make([]int, k)
			copy(cp, current)
			result = append(result, cp)
			return
		}
		for i := start; i < n; i++ {
			next := make([]int, len(current)+1)
			copy(next, current)
			next[len(current)] = i
			gen(i+1, next)
		}
	}
	gen(0, []int{})
	return result
}

// AllCombos generates all word combinations for the given sizes.
// For each size k, it generates all k-element subsets of the words,
// then all permutations of each subset.
func AllCombos(words []string, sizes []int) [][]string {
	n := len(words)
	var combos [][]string
	for _, k := range sizes {
		if k > n {
			continue
		}
		for _, idxs := range Subsets(n, k) {
			pool := make([]string, k)
			for i, idx := range idxs {
				pool[i] = words[idx]
			}
			perms := GeneratePermutations(pool)
			combos = append(combos, perms...)
		}
	}
	return combos
}

// SpecialSubsets generates all combinations of special characters up to maxLen.
// For each subset size k (1..maxLen), it generates all permutations of each
// k-element subset of the character set.
func SpecialSubsets(chars string, maxLen int) []string {
	var result []string
	n := len(chars)
	for k := 1; k <= n && k <= maxLen; k++ {
		for _, idxs := range Subsets(n, k) {
			pool := make([]string, k)
			for i, idx := range idxs {
				pool[i] = string(chars[idx])
			}
			perms := GeneratePermutations(pool)
			for _, p := range perms {
				result = append(result, strings.Join(p, ""))
			}
		}
	}
	return result
}

// ParseNumberRanges parses a string of number ranges into a slice of ints.
// Supported formats: "1-100" (range), "69" (single), "1-10,1900-2030,69" (multiple).
// Reversed ranges (e.g. "5-1") are automatically corrected.
func ParseNumberRanges(s string) ([]int, error) {
	var nums []int
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' '
	})
	for _, p := range parts {
		if p == "" {
			continue
		}
		if strings.Contains(p, "-") {
			halves := strings.SplitN(p, "-", 2)
			if len(halves) != 2 {
				return nil, fmt.Errorf("invalid range: %s", p)
			}
			start, err1 := strconv.Atoi(strings.TrimSpace(halves[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(halves[1]))
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("invalid range: %s", p)
			}
			if start > end {
				start, end = end, start
			}
			for n := start; n <= end; n++ {
				nums = append(nums, n)
			}
		} else {
			n, err := strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", p)
			}
			nums = append(nums, n)
		}
	}
	return nums, nil
}

func readInputWords(reader *bufio.Reader) []string {
	options := []string{
		"Type words directly",
		"Load from a file",
	}
	idx := ui.PromptMenu(reader, "Input source", options, 0)

	var raw []string
	if idx == 0 {
		fmt.Println()
		ui.PrintPrompt("Type words separated by spaces, commas, or new lines.\n")
		ui.PrintPrompt("Enter a blank line to finish:\n")
		for {
			line, err := ui.ReadLine(reader)
			if err != nil {
				ui.PrintError(err.Error())
				return nil
			}
			if line == "" {
				break
			}
			raw = append(raw, strings.FieldsFunc(line, func(r rune) bool {
				return r == ',' || r == ' ' || r == '\t'
			})...)
		}
	} else {
		fmt.Println()
		ui.PrintPrompt("File path: ")
		path, err := ui.ReadLine(reader)
		if err != nil {
			ui.PrintError(err.Error())
			return nil
		}
		if path == "" {
			ui.PrintError("No path given. Exiting.")
			return nil
		}
		if err := ui.ValidateInputPath(path); err != nil {
			ui.PrintError("Invalid file path: " + err.Error())
			return nil
		}
		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			ui.PrintError("Error reading file: " + err.Error())
			return nil
		}
		raw = strings.FieldsFunc(string(data), func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
		})
	}

	return dedupWords(raw)
}

func dedupWords(raw []string) []string {
	seen := make(map[string]bool)
	var words []string
	for _, w := range raw {
		w = strings.TrimSpace(w)
		w = strings.TrimPrefix(w, "\ufeff")
		w = strings.TrimPrefix(w, "\ufffe")
		if w == "" || seen[w] {
			continue
		}
		seen[w] = true
		words = append(words, w)
	}
	return words
}

func chooseDepth(reader *bufio.Reader, n int) []int {
	if n < 2 {
		return []int{1}
	}
	options := []string{
		"1-word only (single words: Apple, Banana)",
		"2-word combos (AppleBanana, BananaApple)",
		"2-word and 3-word combos",
		fmt.Sprintf("All combos 1..%d words (maximum output)", n),
		"Custom (type sizes, e.g. 1,2,3)",
	}
	idx := ui.PromptMenu(reader, "Choose combination depth", options, 2)

	switch idx {
	case 0:
		return []int{1}
	case 1:
		return []int{2}
	case 2:
		return []int{2, 3}
	case 3:
		sizes := make([]int, n)
		for i := 0; i < n; i++ {
			sizes[i] = i + 1
		}
		return sizes
	default:
		fmt.Println()
		ui.PrintPrompt("Enter sizes (comma-separated, e.g. 1,2,3): ")
		line, err := ui.ReadLine(reader)
		if err != nil {
			ui.PrintError(err.Error())
			return []int{2, 3}
		}
		var sizes []int
		for _, p := range strings.Split(line, ",") {
			p = strings.TrimSpace(p)
			k, err := strconv.Atoi(p)
			if err == nil && k >= 1 && k <= n {
				sizes = append(sizes, k)
			}
		}
		if len(sizes) == 0 {
			ui.PrintError("No valid sizes, using 2 and 3.")
			return []int{2, 3}
		}
		return sizes
	}
}

// Display prints the mixer configuration to stdout in a formatted block.
func (c *Config) Display() {
	fmt.Println()
	fmt.Println(ui.Colorize(ui.Yellow+ui.Bold, "Configuration"))
	fmt.Println(ui.Colorize(ui.Yellow, "-------------"))
	fmt.Printf("  Words:            %s\n", ui.Colorize(ui.Cyan, fmt.Sprintf("%d", len(c.Words))))
	fmt.Printf("  Max length cap:   %s\n", ui.Colorize(ui.Cyan, func() string {
		if c.MaxCap > 0 {
			return fmt.Sprintf("%d", c.MaxCap)
		}
		return "none"
	}()))
	fmt.Printf("  Casings:          %s\n", ui.Colorize(ui.Cyan, func() string {
		names := make([]string, len(c.CasingIdxs))
		for i, cs := range c.CasingIdxs {
			names[i] = CasingName(cs)
		}
		return strings.Join(names, ", ")
	}()))
	fmt.Printf("  Append numbers:   %s\n", ui.Colorize(ui.Cyan, func() string {
		if c.AppendNums {
			return fmt.Sprintf("yes (%d numbers)", len(c.Numbers))
		}
		return "no"
	}()))
	fmt.Printf("  Append specials:  %s\n", ui.Colorize(ui.Cyan, func() string {
		if c.AppendSpec {
			return fmt.Sprintf("yes (%d combos, max %d chars)", len(c.SpecCombos), c.MaxSpecLen)
		}
		return "no"
	}()))
	fmt.Printf("  Combo sizes:      %s\n", ui.Colorize(ui.Cyan, fmt.Sprintf("%v", c.Sizes)))
	fmt.Printf("  Total combos:     %s\n", ui.Colorize(ui.Cyan, fmt.Sprintf("%d", len(c.Combos))))
	fmt.Printf("  Max total size:   %s\n", ui.Colorize(ui.Cyan, func() string {
		if c.MaxBytes > 0 {
			return ui.FormatBytes(c.MaxBytes)
		}
		return "no cap"
	}()))
}

// EstimateOutputs calculates the total number of output entries that will
// be generated, based on combos, casings, specials, and numbers.
func (c *Config) EstimateOutputs() uint64 {
	n := uint64(len(c.Combos)) * uint64(len(c.CasingIdxs))
	if c.AppendSpec {
		n *= uint64(len(c.SpecCombos) + 1)
	}
	if c.AppendNums {
		n *= uint64(len(c.Numbers) + 1)
	}
	return n
}

// Generate runs the mixer and writes output to files. Uses stderr for progress.
func (c *Config) Generate(outDir string) (uint64, int, time.Duration, error) {
	return c.GenerateWithProgress(outDir, nil)
}

// GenerateWithProgress runs the mixer with an optional progress callback.
// The progressFn is called every ProgressInterval() words with the current
// count and file number. If progressFn is nil, progress is written to stderr.
// Returns: words written, files created, elapsed time, and any error.
func (c *Config) GenerateWithProgress(outDir string, progressFn func(written uint64, files int)) (uint64, int, time.Duration, error) {
	c.Display()
	fmt.Println()
	fmt.Println(ui.Colorize(ui.Green, "Generating..."))

	start := time.Now()
	fw := ui.NewFileWriter(outDir)

	seen := make(map[string]bool)
	var written uint64
	var stopped bool

	tryWrite := func(out string) {
		if stopped {
			return
		}
		if c.MaxCap > 0 && len(out) > c.MaxCap {
			return
		}
		if seen[out] {
			return
		}
		seen[out] = true
		if err := fw.WriteLine(out); err != nil {
			stopped = true
			return
		}
		written++
		if c.MaxBytes > 0 && fw.CurSize > c.MaxBytes {
			stopped = true
			return
		}
		if written%ui.ProgressInterval() == 0 {
			if progressFn != nil {
				progressFn(written, fw.Files)
			} else {
				fmt.Fprintf(os.Stderr, "\rWritten: %d  File #%d", written, fw.Files)
			}
		}
	}

	for _, combo := range c.Combos {
		for _, casing := range c.CasingIdxs {
			cased := ApplyCasing(combo, casing)
			base := strings.Join(cased, "")

			tryWrite(base)

			if c.AppendSpec {
				for _, sc := range c.SpecCombos {
					tryWrite(base + sc)
				}
			}

			if c.AppendNums {
				for _, n := range c.Numbers {
					nb := base + strconv.Itoa(n)
					tryWrite(nb)
					if c.AppendSpec {
						for _, sc := range c.SpecCombos {
							tryWrite(nb + sc)
						}
					}
				}
			}
		}
	}

	elapsed := time.Since(start)
	fmt.Fprintln(os.Stderr)
	fw.Close()

	if stopped && fw.WriteError != nil {
		return written, fw.Files, elapsed, fw.WriteError
	}
	if stopped && c.MaxBytes > 0 {
		return written, fw.Files, elapsed, fmt.Errorf("stopped at size cap (%s). Wrote %d words", ui.FormatBytes(c.MaxBytes), written)
	}
	return written, fw.Files, elapsed, nil
}

// DedupWords removes duplicate words, strips whitespace and BOM markers.
// This is the exported wrapper around dedupWords for use by external packages.
func DedupWords(raw []string) []string {
	return dedupWords(raw)
}

// Run is the interactive entry point for the word mixer. It collects all
// configuration via interactive prompts, then generates and writes the output.
func Run(reader *bufio.Reader, outDir string, maxBytes uint64) {
	ui.PrintHeader("Word Mixer Mode")

	words := readInputWords(reader)
	if len(words) == 0 {
		ui.PrintError("No words provided. Exiting.")
		return
	}

	fmt.Println()
	fmt.Printf("Loaded %s unique words: %s\n",
		ui.Colorize(ui.Cyan, fmt.Sprintf("%d", len(words))),
		ui.Colorize(ui.White, strings.Join(words, ", ")))

	fmt.Println()
	ui.PrintPrompt("Max length cap (0 = no cap): ")
	capStr, err := ui.ReadLine(reader)
	if err != nil {
		ui.PrintError(err.Error())
		return
	}
	maxCap, err := strconv.Atoi(capStr)
	if err != nil || maxCap < 0 {
		ui.PrintError("Invalid cap, using 0 (no cap).")
		maxCap = 0
	}

	casingOptions := []string{
		"PascalCase (CinoCat)",
		"camelCase (cinoCat)",
		"ALL CAPS (CINOCAT)",
		"all lowercase (cinocat)",
	}
	casingIdxs := ui.PromptMultiMenu(reader, "Choose casing modes (select all you want)", casingOptions, []int{0, 1, 2, 3})
	if len(casingIdxs) == 0 {
		ui.PrintError("No casing selected. Using PascalCase.")
		casingIdxs = []int{0}
	}

	fmt.Println()
	ui.PrintPrompt("Append numbers to words? (y/n) [default: n]: ")
	appendNumsStr, _ := ui.ReadLine(reader)
	appendNums := strings.ToLower(appendNumsStr) == "y" || strings.ToLower(appendNumsStr) == "yes"

	var numbers []int
	if appendNums {
		fmt.Println()
		fmt.Println("Enter number ranges. Examples:")
		fmt.Println("  1-100              -> 1,2,3,...,100")
		fmt.Println("  1900-2030           -> years 1900..2030")
		fmt.Println("  69,420              -> single numbers")
		fmt.Println("  1-10,1900-2030,69   -> multiple ranges combined")
		ui.PrintPrompt("Ranges: ")
		rangesStr, _ := ui.ReadLine(reader)
		numbers, err = ParseNumberRanges(rangesStr)
		if err != nil {
			ui.PrintError("Error parsing ranges: " + err.Error() + " - continuing without numbers")
			appendNums = false
		}
		if len(numbers) == 0 {
			ui.PrintError("No numbers parsed. Continuing without numbers.")
			appendNums = false
		}
	}

	fmt.Println()
	ui.PrintPrompt("Append special characters? (y/n) [default: n]: ")
	appendSpecStr, _ := ui.ReadLine(reader)
	appendSpec := strings.ToLower(appendSpecStr) == "y" || strings.ToLower(appendSpecStr) == "yes"

	var specCombos []string
	maxSpecLen := ui.DefaultMaxSpecLen()
	if appendSpec {
		fmt.Println()
		fmt.Printf("Available special chars: %s\n", ui.Colorize(ui.Yellow, specialChars))
		ui.PrintPrompt("Max special chars to combine (1-8) [default: 4]: ")
		specLenStr, _ := ui.ReadLine(reader)
		specLen, err := strconv.Atoi(specLenStr)
		if err != nil || specLen < 1 || specLen > 8 {
			ui.PrintError("Invalid, using 4.")
			specLen = ui.DefaultMaxSpecLen()
		}
		maxSpecLen = specLen
		specCombos = SpecialSubsets(specialChars, maxSpecLen)
		fmt.Printf("Generated %s special-char combinations.\n", ui.Colorize(ui.Cyan, fmt.Sprintf("%d", len(specCombos))))
	}

	sizes := chooseDepth(reader, len(words))
	combos := AllCombos(words, sizes)

	cfg := &Config{
		Words:      words,
		MaxCap:     maxCap,
		CasingIdxs: casingIdxs,
		AppendNums: appendNums,
		Numbers:    numbers,
		AppendSpec: appendSpec,
		SpecCombos: specCombos,
		MaxSpecLen: maxSpecLen,
		Sizes:      sizes,
		Combos:     combos,
		MaxBytes:   maxBytes,
	}

	if cfg.EstimateOutputs() > ui.LargeOutputThreshold() {
		fmt.Println()
		ui.PrintError(fmt.Sprintf("WARNING: estimated %d total output entries.", cfg.EstimateOutputs()))
		fmt.Println(ui.Colorize(ui.Red, "This will use significant memory for deduplication and may be slow."))
		ui.PrintPrompt("Continue? (y/n) [default: n]: ")
		confirm, _ := ui.ReadLine(reader)
		if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
			fmt.Println("Aborted by user.")
			return
		}
	}

	written, files, elapsed, genErr := cfg.Generate(outDir)
	if genErr != nil {
		fmt.Println()
		ui.PrintError(genErr.Error())
	}
	ui.PrintSummary(written, files, elapsed, outDir)
}