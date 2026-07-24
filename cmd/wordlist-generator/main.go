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

const version = "1.1.0"

func main() {
	mode := flag.String("mode", "", "Mode: 'mixer' or 'numeric'")
	outDir := flag.String("output", "", "Output directory (default: output)")
	maxGB := flag.Float64("max-gb", 0, "Max total output in GB (0 = no cap)")
	words := flag.String("words", "", "Comma-separated words for mixer mode")
	length := flag.Int("length", 0, "String length for numeric mode (1-16)")
	noColor := flag.Bool("no-color", false, "Disable colored output")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("wordlist-generator %s\n", version)
		return
	}

	ui.SetNoColor(*noColor)

	if *mode != "" && *outDir != "" {
		runNonInteractive(*mode, *outDir, *maxGB, *words, *length)
		return
	}

	runInteractive()
}

func runInteractive() {
	m := tui.New()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runNonInteractive(mode, outDir string, maxGB float64, words string, length int) {
	maxBytes := uint64(0)
	if maxGB > 0 {
		maxBytes = uint64(maxGB * float64(1000*1000*1000))
	}

	if err := ui.ValidateOutputPath(outDir); err != nil {
		ui.PrintError("Invalid output path: " + err.Error())
		os.Exit(1)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		ui.PrintError("Error creating output directory: " + err.Error())
		os.Exit(1)
	}

	reader := bufio.NewReader(strings.NewReader(""))

	switch mode {
	case "mixer":
		if words == "" {
			ui.PrintError("--words flag required for mixer mode")
			os.Exit(1)
		}
		runMixerNonInteractive(reader, outDir, maxBytes, words)
	case "numeric":
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

func runMixerNonInteractive(reader *bufio.Reader, outDir string, maxBytes uint64, wordsStr string) {
	raw := strings.FieldsFunc(wordsStr, func(r rune) bool {
		return r == ',' || r == ' '
	})
	words := mixer.DedupWords(raw)
	if len(words) == 0 {
		ui.PrintError("No valid words provided.")
		os.Exit(1)
	}
	fmt.Printf("Words: %s\n", strings.Join(words, ", "))

	casingIdxs := []int{0, 1, 2, 3}
	combos := mixer.AllCombos(words, []int{2, 3})

	cfg := &mixer.Config{
		Words:      words,
		CasingIdxs: casingIdxs,
		Sizes:      []int{2, 3},
		Combos:     combos,
		MaxBytes:   maxBytes,
	}

	written, files, elapsed, err := cfg.Generate(outDir)
	if err != nil {
		ui.PrintError(err.Error())
	}
	ui.PrintSummary(written, files, elapsed, outDir)
}

func runNumericNonInteractive(reader *bufio.Reader, outDir string, maxBytes uint64, length int) {
	if length > 16 {
		ui.PrintError("Length > 16 would produce exabyte-scale output.")
		os.Exit(1)
	}
	numeric.RunNonInteractive(outDir, maxBytes, length)
}