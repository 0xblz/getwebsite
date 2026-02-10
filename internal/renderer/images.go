package renderer

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// supportsInlineImages checks if the terminal supports iTerm2 inline image protocol.
func supportsInlineImages() bool {
	term := os.Getenv("TERM_PROGRAM")
	switch term {
	case "iTerm.app", "WezTerm", "mintty":
		return true
	}
	// Also check LC_TERMINAL for tmux passthrough
	lc := os.Getenv("LC_TERMINAL")
	return lc == "iTerm2"
}

// renderInlineImage fetches an image from url and returns the iTerm2 escape
// sequence to display it inline. Returns empty string on failure.
func renderInlineImage(url string, maxWidth int) string {
	if url == "" {
		return ""
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	// Limit to 5MB
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil || len(data) == 0 {
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	// iTerm2 inline image protocol:
	// ESC ] 1337 ; File=inline=1;width=auto;preserveAspectRatio=1 : <base64> ST
	width := fmt.Sprintf("%d", maxWidth)
	return fmt.Sprintf("\033]1337;File=inline=1;width=%s;preserveAspectRatio=1:%s\a",
		width, encoded)
}
