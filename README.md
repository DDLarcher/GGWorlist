# Wordlist Generator

A terminal-based wordlist generator written in Go. Combines user-provided words with multiple casing modes, number ranges, and special characters — or generates exhaustive numeric combinations.

```
   ___________      __            _____     __
  / ___/ ___/ | /| / /__  _______/ / (_)__ / /_
 / (_ / (_ /| |/ |/ / _ \/ __/ _  / / (_-</ __/
 \___/\___/ |__/|__/\___/_/  \_,_/_/_/___/\__/
                                              
```

## Features

### Word Mixer
- Input words by typing them or loading from a file
- **4 casing modes** (select any combination):
  - PascalCase — `CinoCat`
  - camelCase — `cinoCat`
  - ALL CAPS — `CINOCAT`
  - all lowercase — `cinocat`
- **Combination depth** — choose 1-word, 2-word, 2+3-word, all 1..N, or custom
- **Number appending** — multiple ranges (`1-100`, `1900-2030`, `18`)
- **Special characters** — `! ? $ . @ # * & + =` with combinations of 1–8 chars
- **Max length cap** — skip combos exceeding a character limit
- **Deduplication** — no duplicate output words
- Preserves original word casing (e.g. `ABC` stays `ABC`, not `Abc`)

### Numeric Generator
- Generates **ALL** digit combinations (0–9) of a given length
- Skips any with 3 identical consecutive digits
- Shows estimated count and storage before generating
- Size cap with warning and confirmation

### Both Modes
- **Max total output size** — set a GB cap or leave empty for no limit
- Output split into files of max 1 GB each (`wordlist_001.txt`, `wordlist_002.txt`, ...)
- One word per line
- Colored terminal UI
- Summary box with stats after completion

## Requirements

- **Go 1.21 or newer** — [Download Go](https://go.dev/dl/)

No external dependencies. Uses only the Go standard library.

## Installation

### From source

```bash
git clone https://github.com/<your-username>/wordlist-generator.git
cd wordlist-generator
go build -o wordlist-generator ./cmd/wordlist-generator
```

### Or install directly

```bash
go install ./cmd/wordlist-generator@latest
```

## Usage

### Interactive mode

```bash
./wordlist-generator
```

The program will guide you through an interactive menu:

1. **Choose mode** — Word mixer or Numeric generator
2. **Output directory** — where files are saved (default: `output`)
3. **Max total output in GB** — size cap or empty for no limit
4. Mode-specific options (words, casings, numbers, specials / length)
5. Generation runs with a live progress bar
6. A summary box shows stats and output files when done

### Non-interactive mode (CLI flags)

```bash
# Mixer mode with words, default casings (all 4), 2+3 word combos
./wordlist-generator --mode mixer --output out --words "apple,banana,cherry"

# Numeric mode, length 8, 1 GB cap
./wordlist-generator --mode numeric --output out --length 8 --max-gb 1

# Disable colors (for piping or old terminals)
./wordlist-generator --no-color

# Print version
./wordlist-generator --version
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--mode` | `mixer` or `numeric` (enables non-interactive mode with `--output`) |
| `--output` | Output directory |
| `--words` | Comma-separated words (mixer mode) |
| `--length` | String length 1-16 (numeric mode) |
| `--max-gb` | Max total output in GB (0 = no cap) |
| `--no-color` | Disable ANSI colors |
| `--version` | Print version and exit |

### Example: Word Mixer

```
Mode:           Word mixer
Words:          Daniele Suzanne PQS
Casings:        PascalCase, ALL CAPS
Numbers:        1-100, 1900-2030
Specials:       yes (max 2 chars)
Depth:          2-word and 3-word combos
Max length cap: 30
```

Sample output:
```
DanieleSuzanne
DanieleSuzanne!
SuzannePQS!? 
DanieleSuzannePQS
DANIELESUZANNE2026$
...
```

### Example: Numeric Generator

```
Mode:    Numeric generator
Length:  8
```

Generates all 94,551,300 valid 8-digit combinations (no 3 consecutive identical), ~812 MB.

## Reference: Numeric Generator Output Sizes

| Length | Valid combinations | Approx. storage |
|--------|-------------------|-----------------|
| 4      | 9,810             | 48 KB           |
| 6      | 963,090           | 6.4 MB          |
| 8      | 94,551,300        | 812 MB          |
| 10     | 9,070,162,350     | 90.7 GB         |
| 12     | 868,364,879,250   | 9.6 TB          |

> Lengths above 12 produce petabyte-scale output and are not practical.

## Project Structure

```
wordlist-generator/
├── cmd/
│   └── wordlist-generator/
│       └── main.go            # Entry point, mode selector
├── internal/
│   ├── ui/
│   │   ├── ui.go              # Colors, menus, file writer, summary box
│   │   └── ui_test.go         # UI unit tests
│   ├── mixer/
│   │   ├── mixer.go           # Word mixer logic
│   │   └── mixer_test.go      # Mixer unit tests
│   └── numeric/
│       ├── numeric.go         # Numeric generator logic
│       └── numeric_test.go    # Numeric unit tests
├── go.mod
├── .gitignore
├── CHANGELOG.md
├── LICENSE
└── README.md
```

## Testing

```bash
go test ./... -v
```

Tests cover:
- Casing functions (PascalCase, camelCase, unicode safety)
- Permutation generation (count, uniqueness)
- Subset generation (no slice aliasing)
- Combination generation
- Number range parsing (ranges, single numbers, reversed, errors)
- Word deduplication (including BOM stripping)
- Numeric count validation (brute-force cross-check)
- Byte formatting
- Path traversal validation

## Cross-Platform

Runs on **Windows, Linux, and macOS**. ANSI colors work on all modern terminals (Windows 10+, any Linux/macOS terminal).

### Cross-compiling

```bash
# Linux from Windows
GOOS=linux GOARCH=amd64 go build -o wordlist-generator-linux ./cmd/wordlist-generator

# macOS from Windows
GOOS=darwin GOARCH=amd64 go build -o wordlist-generator-macos ./cmd/wordlist-generator
```

## License

MIT — see [LICENSE](LICENSE).

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/new-option`)
3. Commit your changes (`git commit -m 'Add new option'`)
4. Push to the branch (`git push origin feature/new-option`)
5. Open a Pull Request
