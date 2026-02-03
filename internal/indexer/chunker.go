package indexer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sgx-labs/statelessagent/internal/config"
)

// Chunk represents a portion of a note for embedding.
type Chunk struct {
	Heading string
	Text    string
}

var (
	h2Split = regexp.MustCompile(`(?m)^## `)
	h3Split = regexp.MustCompile(`(?m)^### `)
)

// ChunkByHeadings splits note body by H2 headings, with H3 sub-splitting for large sections.
func ChunkByHeadings(body string) []Chunk {
	parts := h2Split.Split(body, -1)
	var chunks []Chunk

	// First part is intro (before first H2)
	if strings.TrimSpace(parts[0]) != "" {
		chunks = append(chunks, Chunk{Heading: "(intro)", Text: strings.TrimSpace(parts[0])})
	}

	// Find H2 headings for labeling
	headingLocs := h2Split.FindAllStringIndex(body, -1)
	for i, part := range parts[1:] {
		_ = headingLocs // suppress warning
		lines := strings.SplitN(part, "\n", 2)
		heading := strings.TrimSpace(lines[0])
		text := ""
		if len(lines) > 1 {
			text = strings.TrimSpace(lines[1])
		}
		if text == "" {
			continue
		}

		fullText := "## " + heading + "\n" + text

		// If H2 section is too large, try splitting by H3
		if len(fullText) > config.MaxEmbedChars {
			h3Parts := h3Split.Split(fullText, -1)
			if len(h3Parts) > 1 {
				if strings.TrimSpace(h3Parts[0]) != "" {
					chunks = append(chunks, Chunk{
						Heading: heading,
						Text:    strings.TrimSpace(h3Parts[0]),
					})
				}
				for _, h3Part := range h3Parts[1:] {
					h3Lines := strings.SplitN(h3Part, "\n", 2)
					h3Heading := strings.TrimSpace(h3Lines[0])
					h3Text := ""
					if len(h3Lines) > 1 {
						h3Text = strings.TrimSpace(h3Lines[1])
					}
					if h3Text != "" {
						chunks = append(chunks, Chunk{
							Heading: heading + " > " + h3Heading,
							Text:    "### " + h3Heading + "\n" + h3Text,
						})
					}
				}
			} else {
				chunks = append(chunks, Chunk{Heading: heading, Text: fullText})
			}
		} else {
			chunks = append(chunks, Chunk{Heading: heading, Text: fullText})
		}

		_ = i
	}

	if len(chunks) == 0 {
		return []Chunk{{Heading: "(full)", Text: body}}
	}
	return chunks
}

// ChunkBySize splits text into chunks at paragraph boundaries.
func ChunkBySize(text string, maxChars int) []Chunk {
	if maxChars <= 0 {
		maxChars = config.MaxEmbedChars
	}
	paragraphs := strings.Split(text, "\n\n")
	var chunks []Chunk
	var current strings.Builder

	for _, para := range paragraphs {
		if current.Len()+len(para)+2 > maxChars && current.Len() > 0 {
			chunks = append(chunks, Chunk{
				Heading: partHeading(len(chunks) + 1),
				Text:    strings.TrimSpace(current.String()),
			})
			current.Reset()
			current.WriteString(para)
		} else {
			if current.Len() > 0 {
				current.WriteString("\n\n")
			}
			current.WriteString(para)
		}
	}
	if strings.TrimSpace(current.String()) != "" {
		chunks = append(chunks, Chunk{
			Heading: partHeading(len(chunks) + 1),
			Text:    strings.TrimSpace(current.String()),
		})
	}
	return chunks
}

func partHeading(n int) string {
	return fmt.Sprintf("(part %d)", n)
}
