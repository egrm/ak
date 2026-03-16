# ak — Agent-optimized Anki CLI

A minimal CLI that wraps [AnkiConnect](https://foosoft.net/projects/anki-connect/)'s HTTP API. Designed for LLM agents (Claude Code, etc.) but works fine for humans too.

- JSON output by default, `--human` for readable output
- Auto-launches Anki if not running
- Single binary, zero dependencies (Go stdlib only)

## Prerequisites

- macOS (auto-launch uses `open -a Anki`)
- [Go](https://go.dev/) 1.21+ (build only)
- [Anki](https://apps.ankiweb.net/) with [AnkiConnect](https://ankiweb.net/shared/info/2055492159) plugin installed
- AnkiConnect listening on `localhost:8765` (default)

## Install

```bash
go install github.com/egrm/ak@latest
```

Or build from source:

```bash
git clone https://github.com/egrm/ak
cd ak
go build -o ak .
cp ak ~/.local/bin/  # or wherever
```

## Usage

### Create cards

```bash
ak add "What is a compressor?" "A dynamics processor that reduces dynamic range" \
  -d "Music Production" -t mixing,dynamics

ak add -m Cloze "{{c1::Compression ratio}} controls how much gain reduction is applied" \
  -d "Music Production" -t mixing

ak add "What does this waveform show?" "Clipping" \
  --image ./clipping.png -d "Music Production"

ak add -f cards.md  # batch add from file
```

### Search and inspect

```bash
ak search 'deck:"Music Production"'    # notes in a deck (quote names with spaces)
ak search "tag:synthesis"               # by tag
ak search "added:7"                     # added in last 7 days

ak info 1234567890                      # full note details
ak info 111 222 333                     # multiple notes
```

### Manage notes

```bash
ak update 1234567890 -F Front="Updated question"
ak update 1234567890 -F Front="New front" -F Back="New back"

ak tag add 1234567890 newTag1,newTag2
ak tag remove 1234567890 oldTag

ak delete 1234567890
ak delete 111 222 333
```

### Decks and models

```bash
ak decks                    # list all decks
ak decks --stats            # with new/learn/review/total counts
ak deck create "My Deck"    # create (supports :: nesting)

ak models                   # list note types
ak models -m "Basic"        # show fields for a model

ak tags                     # list all tags
```

### Other

```bash
ak browse "tag:synthesis"   # open Anki's browser GUI
ak sync                     # trigger AnkiWeb sync
ak version                  # AnkiConnect version
ak help                     # full command reference
```

## Batch file format

Markdown with `---` separators. A header block sets defaults for all cards:

```markdown
deck: Music Production
tags: synthesis, sound-design
model: Basic
---
Q: What does the cutoff frequency control?
A: The point at which the filter begins attenuating frequencies.
---
Q: What is resonance?
A: Emphasis of frequencies near the cutoff point, creating a peak.
---
model: Cloze
Q: A {{c1::low-pass filter}} removes frequencies above the cutoff.
```

- `Q:` maps to the first field (Front for Basic, Text for Cloze)
- `A:` maps to the second field (Back for Basic, Extra for Cloze)
- Per-card `model:`, `deck:`, `tags:` override the header
- Without `Q:`/`A:` prefixes: first line = front, rest = back

## Output

JSON to stdout by default (agent-optimized). Errors go to stderr as `{"error": "message"}`.

```bash
$ ak add "Question" "Answer" -d "Default"
{
  "id": 1234567890
}

$ ak search "tag:math"
[
  1234567890,
  1234567891
]
```

Use `--human` for readable output:

```bash
$ ak decks --stats --human
Deck              New  Learn  Review  Total
Default             0      0       0      0
Music Production    5      2       8    120
```

## Auto-launch

If Anki isn't running, `ak` launches it automatically and polls AnkiConnect for up to 15 seconds. Transparent to the caller.

## Testing

Tests run against an isolated Anki "Test" profile to protect your real collection:

```bash
# First: create a "Test" profile in Anki (File > Switch Profile > Add)
cd ~/Code/ak
go build -o ak .
bash test.sh
```

The test script switches to the Test profile, runs all tests, and restores your original profile on exit (even on failure/interrupt).
