package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]

	var filtered []string
	for _, a := range args {
		if a == "--human" {
			humanOutput = true
		} else {
			filtered = append(filtered, a)
		}
	}
	args = filtered

	if len(args) == 0 {
		cmdHelp()
		return
	}

	client := NewClient()
	cmd, rest := args[0], args[1:]

	var err error
	switch cmd {
	case "add":
		err = cmdAdd(client, rest)
	case "search":
		err = cmdSearch(client, rest)
	case "info":
		err = cmdInfo(client, rest)
	case "update":
		err = cmdUpdate(client, rest)
	case "delete":
		err = cmdDelete(client, rest)
	case "tag":
		err = cmdTag(client, rest)
	case "tags":
		err = cmdTags(client)
	case "decks":
		err = cmdDecks(client, rest)
	case "deck":
		err = cmdDeckSub(client, rest)
	case "models":
		err = cmdModels(client, rest)
	case "browse":
		err = cmdBrowse(client, rest)
	case "sync":
		err = cmdSync(client)
	case "version":
		err = cmdVersion(client)
	case "help":
		cmdHelp()
	default:
		err = fmt.Errorf("unknown command: %s (run 'ak help' for usage)", cmd)
	}

	if err != nil {
		exitError(err)
	}
}
