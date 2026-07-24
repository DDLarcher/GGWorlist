package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"wordlist-generator/internal/mixer"
	"wordlist-generator/internal/numeric"
	"wordlist-generator/internal/tui"
	"wordlist-generator/internal/ui"
)

// version is the current release version of the wordlist generator
const version = "1.1.0"

// main is the entry point. It parses CLI flags and routes to either
// interactive (bubbletea TUI) or non-interactive (CLI flags) mode.
func main() {
	// Define CLI flags for non-interactive mode
	mode := flag.String("mode", "", "Mode: 'mixer' or 'numeric'")
	outDir := flag.String("output", "", "Output directory (default: output)")
	maxGB := flag.Float64("max-gb", 0, "Max total output in GB (0 = no cap)")
	words := flag.String("words", "", "Comma-separated words for mixer mode")
	length := flag.Int("length", 0, "String length for numeric mode (1-16)")
	noColor := flag.Bool("no-color", false, "Disable colored output")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	// Handle --version flag: print version and exit
	if *showVersion {
		fmt.Printf("wordlist-generator %s\n", version)
		return
	}

	// Disable ANSI colors if --no-color is set
	ui.SetNoColor(*noColor)

	// If --mode and --output are both provided, run in non-interactive mode
	if *mode != "" && *outDir != "" {
		runNonInteractive(*mode, *outDir, *maxGB, *words, *length)
		return
	}

	// Otherwise launch the interactive bubbletea TUI
	runInteractive()
}

// runInteractive starts the bubbletea TUI in full-screen alt-screen mode.
func runInteractive() {
	m := tui.New()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runNonInteractive executes the generator using CLI flags without TUI prompts.
// It validates the output path, creates the directory, and dispatches to the
// appropriate mode handler.
func runNonInteractive(mode, outDir string, maxGB float64, words string, length int) {
	// Convert GB to bytes for the size cap
	maxBytes := uint64(0)
	if maxGB > 0 {
		maxBytes = uint64(maxGB * float64(1000*1000*1000))
	}

	// Validate the output path to prevent path traversal attacks
	if err := ui.ValidateOutputPath(outDir); err != nil {
		ui.PrintError("Invalid output path: " + err.Error())
		os.Exit(1)
	}
	// Create the output directory if it does not exist
	if err := os.MkdirAll(outDir, 0755); err != nil {
		ui.PrintError("Error creating output directory: " + err.Error())
		os.Exit(1)
	}

	// Empty reader since non-interactive mode does not read stdin
	reader := bufio.NewReader(strings.NewReader(""))

	switch mode {
	case "mixer":
		// Mixer mode requires the --words flag
		if words == "" {
			ui.PrintError("--words flag required for mixer mode")
			os.Exit(1)
		}
		runMixerNonInteractive(reader, outDir, maxBytes, words)
	case "numeric":
		// Numeric mode requires the --length flag
		if length < 1 {
			ui.PrintError("--length flag required for numeric mode (1-16)")
			os.Exit(1)
		}
		runNumericNonInteractive(reader, outDir, maxBytes, length)
	default:
		ui.PrintError("Unknown mode: " + mode + " (use 'mixer' or 'numeric')")
		os.Exit(1)
	}
}

// runMixerNonInteractive runs the word mixer using CLI-provided words.
// It splits comma/space-separated words, deduplicates them, generates
// all 2-word and 3-word combinations with all 4 casing modes, and writes
// the output to files.
func runMixerNonInteractive(reader *bufio.Reader, outDir string, maxBytes uint64, wordsStr string) {
	// Split the input string into individual words
	raw := strings.FieldsFunc(wordsStr, func(r rune) bool {
		return r == ',' || r == ' '
	})
	// Deduplicate words (also strips BOM and whitespace)
	words := mixer.DedupWords(raw)
	if len(words) == 0 {
		ui.PrintError("No valid words provided.")
		os.Exit(1)
	}
	fmt.Printf("Words: %s\n", strings.Join(words, ", "))

	// Use all 4 casing modes (PascalCase, camelCase, UPPER, lower)
	casingIdxs := []int{0, 1, 2, 3}
	// Generate combinations of 2-word and 3-word depth
	combos := mixer.AllCombos(words, []int{2, 3})

	// Build the mixer configuration
	cfg := &mixer.Config{
		Words:      words,
		CasingIdxs: casingIdxs,
		Sizes:      []int{2, 3},
		Combos:     combos,
		MaxBytes:   maxBytes,
	}

	// Generate the wordlist and print the summary
	written, files, elapsed, err := cfg.Generate(outDir)
	if err != nil {
		ui.PrintError(err.Error())
	}
	ui.PrintSummary(written, files, elapsed, outDir)
}

// runNumericNonInteractive runs the numeric generator with the given length.
// It rejects lengths above 16 to prevent exabyte-scale output.
func runNumericNonInteractive(reader *bufio.Reader, outDir string, maxBytes uint64, length int) {
	if length > 16 {
		ui.PrintError("Length > 16 would produce exabyte-scale output.")
		os.Exit(1)
	}
	numeric.RunNonInteractive(outDir, maxBytes, length)
}