package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Types

type NoteInfo struct {
	ID         int64             `json:"id"`
	Model      string            `json:"model"`
	Tags       []string          `json:"tags"`
	Fields     map[string]string `json:"fields"`
	FieldOrder []string          `json:"-"`
	Cards      []int64           `json:"cards"`
}

type DeckStat struct {
	Name   string `json:"name"`
	New    int    `json:"new"`
	Learn  int    `json:"learn"`
	Review int    `json:"review"`
	Total  int    `json:"total"`
}

// ak version

func cmdVersion(c *Client) error {
	result, err := c.Call("version", nil)
	if err != nil {
		return err
	}
	if humanOutput {
		var v int
		json.Unmarshal(result, &v)
		fmt.Printf("AnkiConnect v%d\n", v)
		return nil
	}
	output(map[string]json.RawMessage{"version": result})
	return nil
}

// ak decks [--stats]

func cmdDecks(c *Client, args []string) error {
	for _, a := range args {
		if a == "--stats" {
			return cmdDeckStats(c)
		}
	}

	result, err := c.Call("deckNames", nil)
	if err != nil {
		return err
	}
	if humanOutput {
		var decks []string
		json.Unmarshal(result, &decks)
		for _, d := range decks {
			fmt.Println(d)
		}
		return nil
	}
	outputRaw(result)
	return nil
}

func cmdDeckStats(c *Client) error {
	namesRaw, err := c.Call("deckNames", nil)
	if err != nil {
		return err
	}
	var names []string
	json.Unmarshal(namesRaw, &names)

	result, err := c.Call("getDeckStats", map[string]any{"decks": names})
	if err != nil {
		return err
	}

	var statsMap map[string]struct {
		Name        string `json:"name"`
		NewCount    int    `json:"new_count"`
		LearnCount  int    `json:"learn_count"`
		ReviewCount int    `json:"review_count"`
		TotalInDeck int    `json:"total_in_deck"`
	}
	json.Unmarshal(result, &statsMap)

	var stats []DeckStat
	for _, s := range statsMap {
		stats = append(stats, DeckStat{
			Name:   s.Name,
			New:    s.NewCount,
			Learn:  s.LearnCount,
			Review: s.ReviewCount,
			Total:  s.TotalInDeck,
		})
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Name < stats[j].Name })

	if humanOutput {
		printDeckStatsHuman(stats)
		return nil
	}
	output(stats)
	return nil
}

// ak deck create "Name"

func cmdDeckSub(c *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ak deck create <name>")
	}
	if args[0] != "create" {
		return fmt.Errorf("unknown deck subcommand: %s", args[0])
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: ak deck create <name>")
	}
	return cmdDeckCreate(c, args[1])
}

func cmdDeckCreate(c *Client, name string) error {
	result, err := c.Call("createDeck", map[string]any{"deck": name})
	if err != nil {
		return err
	}
	if humanOutput {
		var id int64
		json.Unmarshal(result, &id)
		fmt.Printf("Created deck: %s (id: %d)\n", name, id)
		return nil
	}
	var id int64
	json.Unmarshal(result, &id)
	output(map[string]any{"id": id, "name": name})
	return nil
}

// ak models [-m "ModelName"]

func cmdModels(c *Client, args []string) error {
	var modelName string
	for i := 0; i < len(args); i++ {
		if args[i] == "-m" && i+1 < len(args) {
			i++
			modelName = args[i]
		}
	}

	if modelName != "" {
		return cmdModelFields(c, modelName)
	}

	result, err := c.Call("modelNames", nil)
	if err != nil {
		return err
	}
	if humanOutput {
		var models []string
		json.Unmarshal(result, &models)
		for _, m := range models {
			fmt.Println(m)
		}
		return nil
	}
	outputRaw(result)
	return nil
}

func cmdModelFields(c *Client, model string) error {
	result, err := c.Call("modelFieldNames", map[string]any{"modelName": model})
	if err != nil {
		return err
	}
	if humanOutput {
		var fields []string
		json.Unmarshal(result, &fields)
		fmt.Printf("Fields for %s:\n", model)
		for _, f := range fields {
			fmt.Printf("  %s\n", f)
		}
		return nil
	}
	outputRaw(result)
	return nil
}

// ak tags

func cmdTags(c *Client) error {
	result, err := c.Call("getTags", nil)
	if err != nil {
		return err
	}
	if humanOutput {
		var tags []string
		json.Unmarshal(result, &tags)
		if len(tags) == 0 {
			fmt.Println("No tags")
			return nil
		}
		for _, t := range tags {
			fmt.Println(t)
		}
		return nil
	}
	outputRaw(result)
	return nil
}

// ak sync

func cmdSync(c *Client) error {
	_, err := c.Call("sync", nil)
	if err != nil {
		return err
	}
	if humanOutput {
		fmt.Println("Synced")
		return nil
	}
	output(map[string]string{"status": "ok"})
	return nil
}

// ak add

func cmdAdd(c *Client, args []string) error {
	var deck, model, file, imagePath, audioPath string
	var imageField, audioField string
	var tags string
	var positional []string

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "-d" || args[i] == "--deck":
			i++
			if i >= len(args) {
				return fmt.Errorf("-d requires a deck name")
			}
			deck = args[i]
		case args[i] == "-m" || args[i] == "--model":
			i++
			if i >= len(args) {
				return fmt.Errorf("-m requires a model name")
			}
			model = args[i]
		case args[i] == "-t" || args[i] == "--tags":
			i++
			if i >= len(args) {
				return fmt.Errorf("-t requires tags")
			}
			tags = args[i]
		case args[i] == "-f" || args[i] == "--file":
			i++
			if i >= len(args) {
				return fmt.Errorf("-f requires a file path")
			}
			file = args[i]
		case args[i] == "--image" || strings.HasPrefix(args[i], "--image:"):
			if idx := strings.Index(args[i], ":"); idx != -1 {
				imageField = args[i][idx+1:]
			}
			i++
			if i >= len(args) {
				return fmt.Errorf("--image requires a file path")
			}
			imagePath = args[i]
		case args[i] == "--audio" || strings.HasPrefix(args[i], "--audio:"):
			if idx := strings.Index(args[i], ":"); idx != -1 {
				audioField = args[i][idx+1:]
			}
			i++
			if i >= len(args) {
				return fmt.Errorf("--audio requires a file path")
			}
			audioPath = args[i]
		default:
			positional = append(positional, args[i])
		}
	}

	if deck == "" {
		deck = "Default"
	}
	if model == "" {
		model = "Basic"
	}

	if file != "" {
		return cmdAddBatch(c, file, deck, model, tags)
	}

	if len(positional) == 0 {
		return fmt.Errorf("usage: ak add <front> [back] [-d deck] [-t tags] [-m model]")
	}

	fieldNames, err := getModelFields(c, model)
	if err != nil {
		return err
	}

	fields := map[string]string{}
	if len(fieldNames) > 0 {
		fields[fieldNames[0]] = positional[0]
	}
	if len(positional) > 1 && len(fieldNames) > 1 {
		fields[fieldNames[1]] = positional[1]
	}

	// Media
	if imagePath != "" {
		field := imageField
		if field == "" && len(fieldNames) > 1 {
			field = fieldNames[1]
		}
		tag, err := storeMedia(c, imagePath, "image")
		if err != nil {
			return fmt.Errorf("store image: %w", err)
		}
		fields[field] = fields[field] + tag
	}
	if audioPath != "" {
		field := audioField
		if field == "" && len(fieldNames) > 1 {
			field = fieldNames[1]
		}
		tag, err := storeMedia(c, audioPath, "audio")
		if err != nil {
			return fmt.Errorf("store audio: %w", err)
		}
		fields[field] = fields[field] + tag
	}

	var tagList []string
	if tags != "" {
		tagList = parseTags(tags)
	}

	note := map[string]any{
		"deckName":  deck,
		"modelName": model,
		"fields":    fields,
		"tags":      tagList,
		"options": map[string]any{
			"allowDuplicate": false,
		},
	}

	result, err := c.Call("addNote", map[string]any{"note": note})
	if err != nil {
		return err
	}

	var id int64
	json.Unmarshal(result, &id)

	if humanOutput {
		fmt.Printf("Created note %d\n", id)
		return nil
	}
	output(map[string]any{"id": id})
	return nil
}

func cmdAddBatch(c *Client, file, defaultDeck, defaultModel, defaultTags string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	cards := parseBatchFile(string(data), defaultDeck, defaultModel, defaultTags)
	if len(cards) == 0 {
		return fmt.Errorf("no cards found in file")
	}

	fieldCache := map[string][]string{}
	var notes []map[string]any

	for _, card := range cards {
		fields, ok := fieldCache[card.Model]
		if !ok {
			fields, err = getModelFields(c, card.Model)
			if err != nil {
				return fmt.Errorf("model %q: %w", card.Model, err)
			}
			fieldCache[card.Model] = fields
		}

		noteFields := map[string]string{}
		if len(fields) > 0 {
			noteFields[fields[0]] = card.Q
		}
		if len(fields) > 1 && card.A != "" {
			noteFields[fields[1]] = card.A
		}

		notes = append(notes, map[string]any{
			"deckName":  card.Deck,
			"modelName": card.Model,
			"fields":    noteFields,
			"tags":      card.Tags,
			"options": map[string]any{
				"allowDuplicate": false,
			},
		})
	}

	result, err := c.Call("addNotes", map[string]any{"notes": notes})
	if err != nil {
		return err
	}

	if humanOutput {
		var ids []any
		json.Unmarshal(result, &ids)
		created := 0
		for _, id := range ids {
			if id != nil {
				created++
			}
		}
		fmt.Printf("Created %d/%d notes\n", created, len(cards))
		return nil
	}
	outputRaw(result)
	return nil
}

// ak search "query"

func cmdSearch(c *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ak search <query>")
	}
	query := strings.Join(args, " ")

	result, err := c.Call("findNotes", map[string]any{"query": query})
	if err != nil {
		return err
	}

	if humanOutput {
		var ids []int64
		json.Unmarshal(result, &ids)
		if len(ids) == 0 {
			fmt.Println("No results")
			return nil
		}
		for _, id := range ids {
			fmt.Println(id)
		}
		return nil
	}
	outputRaw(result)
	return nil
}

// ak info <id> [<id>...]

func cmdInfo(c *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ak info <note-id> [<note-id>...]")
	}

	var ids []int64
	for _, a := range args {
		id, err := strconv.ParseInt(a, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid note ID: %s", a)
		}
		ids = append(ids, id)
	}

	result, err := c.Call("notesInfo", map[string]any{"notes": ids})
	if err != nil {
		return err
	}

	var rawNotes []struct {
		NoteID    int64    `json:"noteId"`
		ModelName string   `json:"modelName"`
		Tags      []string `json:"tags"`
		Fields    map[string]struct {
			Value string `json:"value"`
			Order int    `json:"order"`
		} `json:"fields"`
		Cards []int64 `json:"cards"`
	}
	json.Unmarshal(result, &rawNotes)

	var notes []NoteInfo
	for _, rn := range rawNotes {
		if rn.NoteID == 0 {
			continue
		}
		note := NoteInfo{
			ID:     rn.NoteID,
			Model:  rn.ModelName,
			Tags:   rn.Tags,
			Cards:  rn.Cards,
			Fields: make(map[string]string),
		}
		if note.Tags == nil {
			note.Tags = []string{}
		}

		type fieldEntry struct {
			name  string
			order int
		}
		var ordered []fieldEntry
		for name, f := range rn.Fields {
			note.Fields[name] = f.Value
			ordered = append(ordered, fieldEntry{name, f.Order})
		}
		sort.Slice(ordered, func(i, j int) bool { return ordered[i].order < ordered[j].order })
		for _, f := range ordered {
			note.FieldOrder = append(note.FieldOrder, f.name)
		}

		notes = append(notes, note)
	}

	if len(notes) == 0 {
		return fmt.Errorf("no notes found for given ID(s)")
	}

	if humanOutput {
		for i, note := range notes {
			if i > 0 {
				fmt.Println()
			}
			printNoteHuman(note)
		}
		return nil
	}
	output(notes)
	return nil
}

// ak update <id> -F Field="value"

func cmdUpdate(c *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ak update <note-id> -F Field=\"value\"")
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid note ID: %s", args[0])
	}

	fields := map[string]string{}
	for i := 1; i < len(args); i++ {
		if args[i] == "-F" && i+1 < len(args) {
			i++
			parts := strings.SplitN(args[i], "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid field format: %s (expected Field=value)", args[i])
			}
			fields[parts[0]] = parts[1]
		}
	}

	if len(fields) == 0 {
		return fmt.Errorf("no fields specified (use -F Field=\"value\")")
	}

	_, err = c.Call("updateNoteFields", map[string]any{
		"note": map[string]any{
			"id":     id,
			"fields": fields,
		},
	})
	if err != nil {
		return err
	}

	if humanOutput {
		fmt.Printf("Updated note %d\n", id)
		return nil
	}
	output(map[string]any{"id": id, "status": "updated"})
	return nil
}

// ak delete <id> [<id>...]

func cmdDelete(c *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ak delete <note-id> [<note-id>...]")
	}

	var ids []int64
	for _, a := range args {
		id, err := strconv.ParseInt(a, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid note ID: %s", a)
		}
		ids = append(ids, id)
	}

	_, err := c.Call("deleteNotes", map[string]any{"notes": ids})
	if err != nil {
		return err
	}

	if humanOutput {
		fmt.Printf("Deleted %d note(s)\n", len(ids))
		return nil
	}
	output(map[string]any{"deleted": ids})
	return nil
}

// ak tag add/remove <id> tag1,tag2

func cmdTag(c *Client, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: ak tag add|remove <note-id> <tags>")
	}

	action := args[0]
	id, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid note ID: %s", args[1])
	}

	// Convert comma-separated to space-separated for AnkiConnect
	tagStr := strings.ReplaceAll(args[2], ",", " ")

	switch action {
	case "add":
		_, err = c.Call("addTags", map[string]any{
			"notes": []int64{id},
			"tags":  tagStr,
		})
	case "remove":
		_, err = c.Call("removeTags", map[string]any{
			"notes": []int64{id},
			"tags":  tagStr,
		})
	default:
		return fmt.Errorf("unknown tag subcommand: %s (use add or remove)", action)
	}
	if err != nil {
		return err
	}

	if humanOutput {
		fmt.Printf("Tags %sd on note %d\n", action, id)
		return nil
	}
	output(map[string]any{"id": id, "action": action, "tags": tagStr})
	return nil
}

// ak browse "query"

func cmdBrowse(c *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ak browse <query>")
	}
	query := strings.Join(args, " ")

	_, err := c.Call("guiBrowse", map[string]any{"query": query})
	if err != nil {
		return err
	}

	if humanOutput {
		fmt.Println("Opened browser")
		return nil
	}
	output(map[string]string{"status": "ok", "query": query})
	return nil
}

// Helpers

func getModelFields(c *Client, model string) ([]string, error) {
	result, err := c.Call("modelFieldNames", map[string]any{"modelName": model})
	if err != nil {
		return nil, err
	}
	var fields []string
	json.Unmarshal(result, &fields)
	return fields, nil
}

// ak help

func cmdHelp() {
	fmt.Print(helpText)
}

const helpText = `ak — Agent-optimized Anki CLI

USAGE
  ak <command> [flags] [args...]

GLOBAL FLAGS
  --human    Human-readable output (default: JSON)

COMMANDS
  add        Create notes
  search     Find notes by query
  info       Get note details
  update     Update note fields
  delete     Delete notes
  tag        Manage tags on notes
  tags       List all tags
  decks      List decks
  deck       Deck management
  models     List note types
  browse     Open Anki browser
  sync       Trigger AnkiWeb sync
  version    Show AnkiConnect version
  help       Show this help

ADD
  ak add <front> [back] [-d deck] [-t tags] [-m model] [--image path] [--audio path]
  ak add -f <file>

  -d, --deck     Deck name (default: "Default")
  -m, --model    Note type (default: "Basic")
  -t, --tags     Comma-separated tags
  -f, --file     Batch add from markdown file
  --image        Attach image (appended to Back field)
  --image:Field  Attach image to specific field
  --audio        Attach audio (appended to Back field)
  --audio:Field  Attach audio to specific field

  Cloze: ak add -m Cloze "The {{c1::answer}} is here"

SEARCH
  ak search <query>

  Uses Anki search syntax: deck:Name, tag:name, added:N, rated:N, etc.
  Full syntax: https://docs.ankiweb.net/searching.html

INFO
  ak info <note-id> [<note-id>...]

  Returns full note details: fields, tags, model, cards.

UPDATE
  ak update <note-id> -F Field="value" [-F Field="value"]

DELETE
  ak delete <note-id> [<note-id>...]

TAGS
  ak tag add <note-id> <tags>     Add comma-separated tags
  ak tag remove <note-id> <tags>  Remove comma-separated tags
  ak tags                         List all tags

DECKS
  ak decks              List all decks
  ak decks --stats      List decks with card counts
  ak deck create <name> Create a deck (supports :: nesting)

MODELS
  ak models             List all note types
  ak models -m <name>   Show fields for a note type

OTHER
  ak browse <query>     Open Anki browser with query
  ak sync               Trigger AnkiWeb sync
  ak version            Show AnkiConnect version

BATCH FILE FORMAT
  Markdown with --- separators. Header block sets defaults:

  deck: Music Production
  tags: topic1, topic2
  model: Basic
  ---
  Q: Question text
  A: Answer text
  ---
  model: Cloze
  Q: The {{c1::answer}} is here

OUTPUT
  Default: JSON to stdout. Errors: JSON to stderr.
  Use --human for human-readable output.

AUTO-LAUNCH
  If Anki is not running, ak launches it and waits up to 15s.
`
