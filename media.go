package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

func storeMedia(c *Client, path, mediaType string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	filename := filepath.Base(path)
	encoded := base64.StdEncoding.EncodeToString(data)

	_, err = c.Call("storeMediaFile", map[string]any{
		"filename": filename,
		"data":     encoded,
	})
	if err != nil {
		return "", err
	}

	switch mediaType {
	case "image":
		return fmt.Sprintf(`<img src="%s">`, filename), nil
	case "audio":
		return fmt.Sprintf("[sound:%s]", filename), nil
	default:
		return "", fmt.Errorf("unknown media type: %s", mediaType)
	}
}
