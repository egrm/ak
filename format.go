package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

var humanOutput bool

func output(v any) {
	if humanOutput {
		printHuman(v)
		return
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		exitError(err)
	}
}

func outputRaw(data json.RawMessage) {
	if humanOutput {
		var v any
		if err := json.Unmarshal(data, &v); err == nil {
			printHuman(v)
			return
		}
	}
	var buf bytes.Buffer
	json.Indent(&buf, data, "", "  ")
	buf.WriteTo(os.Stdout)
	fmt.Println()
}

func exitError(err error) {
	data, _ := json.Marshal(map[string]string{"error": err.Error()})
	fmt.Fprintln(os.Stderr, string(data))
	os.Exit(1)
}

func printHuman(v any) {
	switch val := v.(type) {
	case string:
		fmt.Println(val)
	case []string:
		for _, s := range val {
			fmt.Println(s)
		}
	case []any:
		for _, item := range val {
			fmt.Println(item)
		}
	case float64:
		if val == float64(int64(val)) {
			fmt.Printf("%d\n", int64(val))
		} else {
			fmt.Println(val)
		}
	default:
		data, _ := json.MarshalIndent(v, "", "  ")
		fmt.Println(string(data))
	}
}

func printNoteHuman(note NoteInfo) {
	fmt.Printf("Note %d\n", note.ID)
	fmt.Printf("Model: %s\n", note.Model)
	if len(note.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(note.Tags, ", "))
	}
	fmt.Println("---")
	for _, name := range note.FieldOrder {
		fmt.Printf("%s: %s\n", name, note.Fields[name])
	}
}

func printDeckStatsHuman(stats []DeckStat) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Deck\tNew\tLearn\tReview\tTotal")
	for _, s := range stats {
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\n", s.Name, s.New, s.Learn, s.Review, s.Total)
	}
	w.Flush()
}
