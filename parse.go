package main

import (
	"strings"
)

type CardSpec struct {
	Model string
	Deck  string
	Tags  []string
	Q     string
	A     string
}

func parseBatchFile(content, defaultDeck, defaultModel, defaultTags string) []CardSpec {
	sections := splitSections(content)
	if len(sections) == 0 {
		return nil
	}

	headerDeck, headerModel, headerTags := defaultDeck, defaultModel, parseTags(defaultTags)

	// Check if first section is a header (no Q:/A: content)
	first := strings.TrimSpace(sections[0])
	isHeader := first != "" && !hasCardContent(first)

	startIdx := 0
	if isHeader {
		for _, line := range strings.Split(first, "\n") {
			line = strings.TrimSpace(line)
			if k, v, ok := parseHeaderLine(line); ok {
				switch k {
				case "deck":
					headerDeck = v
				case "model":
					headerModel = v
				case "tags":
					headerTags = parseTags(v)
				}
			}
		}
		startIdx = 1
	}

	var cards []CardSpec
	for _, section := range sections[startIdx:] {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		card := CardSpec{
			Deck:  headerDeck,
			Model: headerModel,
			Tags:  append([]string{}, headerTags...),
		}

		state := "none" // none, q, a
		var contentLines []string

		for _, line := range strings.Split(section, "\n") {
			trimmed := strings.TrimSpace(line)

			if k, v, ok := parseHeaderLine(trimmed); ok {
				switch k {
				case "model":
					card.Model = v
				case "deck":
					card.Deck = v
				case "tags":
					card.Tags = parseTags(v)
				}
				continue
			}

			if strings.HasPrefix(trimmed, "Q:") {
				card.Q = strings.TrimSpace(strings.TrimPrefix(trimmed, "Q:"))
				state = "q"
				continue
			}
			if strings.HasPrefix(trimmed, "A:") {
				card.A = strings.TrimSpace(strings.TrimPrefix(trimmed, "A:"))
				state = "a"
				continue
			}

			switch state {
			case "q":
				card.Q += "\n" + line
			case "a":
				card.A += "\n" + line
			default:
				contentLines = append(contentLines, line)
			}
		}

		// Fallback: no Q:/A: prefix — first line is Q, rest is A
		if state == "none" && len(contentLines) > 0 {
			card.Q = strings.TrimSpace(contentLines[0])
			if len(contentLines) > 1 {
				card.A = strings.TrimSpace(strings.Join(contentLines[1:], "\n"))
			}
		}

		card.Q = strings.TrimSpace(card.Q)
		card.A = strings.TrimSpace(card.A)

		if card.Q != "" {
			cards = append(cards, card)
		}
	}

	return cards
}

func splitSections(content string) []string {
	lines := strings.Split(content, "\n")
	var sections []string
	var current []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			sections = append(sections, strings.Join(current, "\n"))
			current = nil
		} else {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		sections = append(sections, strings.Join(current, "\n"))
	}
	return sections
}

func hasCardContent(section string) bool {
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Q:") || strings.HasPrefix(trimmed, "A:") {
			return true
		}
	}
	return false
}

func parseHeaderLine(line string) (key, value string, ok bool) {
	lower := strings.ToLower(line)
	for _, prefix := range []string{"deck:", "model:", "tags:"} {
		if strings.HasPrefix(lower, prefix) {
			return strings.TrimSuffix(prefix, ":"), strings.TrimSpace(line[len(prefix):]), true
		}
	}
	return "", "", false
}

func parseTags(s string) []string {
	if s == "" {
		return nil
	}
	var tags []string
	for _, t := range strings.Split(s, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}
