package renderer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"time"

	_ "golang.org/x/image/webp"

	"github.com/qeesung/image2ascii/convert"
)

// supportsInlineImages checks if the terminal supports iTerm2 inline image protocol.
func supportsInlineImages() bool {
	term := os.Getenv("TERM_PROGRAM")
	switch term {
	case "iTerm.app", "WezTerm", "mintty":
		return true
	}
	lc := os.Getenv("LC_TERMINAL")
	return lc == "iTerm2"
}

// fetchImage downloads an image and returns the raw bytes.
func fetchImage(url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("empty url")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, err
	}
	return data, nil
}

// renderInlineImage returns the iTerm2 escape sequence to display an image inline.
func renderInlineImage(url string, maxWidth int) string {
	data, err := fetchImage(url)
	if err != nil || len(data) == 0 {
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	width := fmt.Sprintf("%d", maxWidth)
	return fmt.Sprintf("\033]1337;File=inline=1;width=%s;preserveAspectRatio=1:%s\a",
		width, encoded)
}

// renderASCIIImage fetches an image and converts it to ASCII art.
func renderASCIIImage(url string, maxWidth int) string {
	data, err := fetchImage(url)
	if err != nil || len(data) == 0 {
		return ""
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ""
	}

	converter := convert.NewImageConverter()
	opts := convert.DefaultOptions
	opts.FixedWidth = maxWidth
	opts.FixedHeight = 0 // auto based on aspect ratio
	opts.Colored = true

	return converter.Image2ASCIIString(img, &opts)
}
