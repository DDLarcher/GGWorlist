# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-07-23

### Added
- Word mixer mode with 4 casing modes (PascalCase, camelCase, ALL CAPS, lowercase)
- Numeric generator mode (all digit combos of length n, no 3 consecutive)
- Combination depth selector (1-word, 2-word, 2+3-word, all 1..N, custom)
- Number range appending (e.g. `1-100,1900-2030,69`)
- Special character combinations (`!?$.@#*&+=`, up to 8 chars)
- Max length cap for mixed words
- Max total output size cap (GB)
- Colored terminal UI with ANSI codes
- `--no-color` flag to disable colors
- `--version` flag
- CLI flags for non-interactive mode (`--mode`, `--output`, `--words`, `--length`, `--max-gb`)
- Path traversal protection for output directory and input files
- File overwrite warning
- Write error detection and abort
- Unicode-safe casing (UTF-8 multi-byte chars)
- BOM stripping from input files
- Input file size limit (100 MB)
- Memory warning for large mixer outputs
- Boxed summary with stats and file listing
- Unit tests for all pure functions
- MIT license

### Security
- Output path validation (rejects `..` and paths outside CWD/home)
- Input file path validation
- Overflow-safe total bytes calculation in numeric generator
- Write error tracking in file writer
- EOF detection on input