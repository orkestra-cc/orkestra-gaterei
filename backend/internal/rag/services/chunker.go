package services

import (
	"regexp"
	"strings"
)

// Chunk represents a text segment from a document
type Chunk struct {
	Text         string
	Position     int
	SectionTitle string
}

// ChunkText splits text into overlapping chunks, detecting section headings
func ChunkText(text string, chunkSize, overlap int) []Chunk {
	if chunkSize <= 0 {
		chunkSize = 512
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 4
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// Split into paragraphs
	paragraphs := splitParagraphs(text)

	var chunks []Chunk
	var current strings.Builder
	currentSection := ""
	position := 0

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// Detect section headings
		if heading := detectHeading(para); heading != "" {
			currentSection = heading
		}

		// Would adding this paragraph exceed chunk size?
		if current.Len() > 0 && current.Len()+len(para)+1 > chunkSize {
			// Emit current chunk
			chunks = append(chunks, Chunk{
				Text:         strings.TrimSpace(current.String()),
				Position:     position,
				SectionTitle: currentSection,
			})
			position++

			// Start new chunk with overlap from end of previous
			prev := current.String()
			current.Reset()
			if overlap > 0 && len(prev) > overlap {
				current.WriteString(prev[len(prev)-overlap:])
				current.WriteString("\n\n")
			}
		}

		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
	}

	// Emit final chunk
	if current.Len() > 0 {
		chunks = append(chunks, Chunk{
			Text:         strings.TrimSpace(current.String()),
			Position:     position,
			SectionTitle: currentSection,
		})
	}

	return chunks
}

func splitParagraphs(text string) []string {
	// Split on double newlines or more
	re := regexp.MustCompile(`\n\s*\n`)
	return re.Split(text, -1)
}

// ISO clause/section heading patterns
var headingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^(?:Clause|Section|Article)\s+\d+(\.\d+)*`),
	regexp.MustCompile(`^\d+(\.\d+)*\s+[A-Z]`),                  // "4.1 Context of the organization"
	regexp.MustCompile(`^[A-Z][A-Z\s]{3,}$`),                     // ALL CAPS headings
	regexp.MustCompile(`^Annex\s+[A-Z]`),                         // "Annex A"
	regexp.MustCompile(`^(?:Introduction|Scope|Normative references|Terms and definitions)`),
}

func detectHeading(para string) string {
	firstLine := strings.SplitN(para, "\n", 2)[0]
	firstLine = strings.TrimSpace(firstLine)

	for _, pattern := range headingPatterns {
		if pattern.MatchString(firstLine) {
			return firstLine
		}
	}
	return ""
}
