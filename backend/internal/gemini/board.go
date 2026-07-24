package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"

	"theresa/backend/internal/models"
)

const systemInstruction = `You are a patient, clear tutor. Teach the user's question or
document step by step, breaking the explanation into a sequence of "boards" suitable for
a visual teaching board. Use standard English only - no slang, no persona, no regional
dialect.

Respond with ONLY a JSON array of boards, shown to the student one after another. Each
board is an object with:
- "kind": either "lines" (typed prose/math/code) or "diagram" (a Mermaid diagram)
- "title": a short optional heading for this board - a few words at most, never a full
  sentence or the explanation itself. The explanation always belongs in "lines".
- "lines": (for kind "lines") REQUIRED, an array of strings, each a line of the
  explanation. A line may contain inline math wrapped in single dollar signs ($...$),
  inline code wrapped in backticks (` + "`...`" + `), or be an entire fenced code block (using
  triple backticks). Keep each board's lines focused on one idea - not a wall of text. Every
  math expression must appear exactly once, fully resolved - never repeat a partially-worked
  step across multiple lines.
- "mermaid": (for kind "diagram") Mermaid diagram syntax - flowcharts or sequence diagrams
  for cycles, branches, or sequences of steps. Never use a diagram for a numeric graph or
  plot with axes - Mermaid cannot render that meaningfully. Describe a graph/plot in words
  via "lines" instead.

Do not include any text outside the JSON array.`

// boardUnitSchema describes one board (an object discriminated by "kind"),
// used both as the array item schema for M3's one-shot response below and
// conceptually mirrored (as two separate per-tool schemas) by M4's live
// show_working/draw_diagram tools.
var boardUnitSchema = &genai.Schema{
	Type:     genai.TypeObject,
	Required: []string{"kind"},
	Properties: map[string]*genai.Schema{
		"kind":    {Type: genai.TypeString, Enum: []string{"lines", "diagram"}},
		"title":   {Type: genai.TypeString},
		"lines":   {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
		"mermaid": {Type: genai.TypeString},
	},
}

var boardResponseSchema = &genai.Schema{
	Type:  genai.TypeArray,
	Items: boardUnitSchema,
}

type BoardRequest struct {
	Text         string
	FileURI      string
	FileMimeType string
}

// GenerateBoard asks Gemini to teach req.Text (optionally grounded in an
// uploaded file) and returns an ordered list of boards. If Gemini's response
// isn't valid JSON, it's wrapped as a single lines-board rather than failing
// the request.
func (c *Client) GenerateBoard(ctx context.Context, req BoardRequest) ([]models.BoardContent, error) {
	parts := []*genai.Part{{Text: req.Text}}
	if req.FileURI != "" {
		parts = append(parts, &genai.Part{FileData: &genai.FileData{FileURI: req.FileURI, MIMEType: req.FileMimeType}})
	}
	content := genai.NewContentFromParts(parts, genai.RoleUser)

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, genai.RoleUser),
		ResponseMIMEType:  "application/json",
		ResponseSchema:    boardResponseSchema,
	}

	resp, err := c.genai.Models.GenerateContent(ctx, c.textModel, []*genai.Content{content}, config)
	if err != nil {
		return nil, fmt.Errorf("generate board: %w", err)
	}

	text := resp.Text()

	var boards []models.BoardContent
	if err := json.Unmarshal([]byte(text), &boards); err != nil || len(boards) == 0 {
		return []models.BoardContent{{Kind: "lines", Lines: sanitizeFallbackText(text)}}, nil
	}

	usable := boards[:0]
	for i := range boards {
		if normalizeBoard(&boards[i]) {
			usable = append(usable, boards[i])
		}
	}

	if len(usable) == 0 {
		return []models.BoardContent{{Kind: "lines", Lines: sanitizeFallbackText(text)}}, nil
	}

	return usable, nil
}

const maxFallbackChars = 600

// sanitizeFallbackText turns raw non-JSON model output into a few
// board-sized lines instead of one giant unreadable blob. Protects both the
// Board's character-by-character typewriter reveal (a multi-thousand-
// character single line would take minutes) and anything deriving a title
// from board content, against a degenerate/repetitive model response -
// observed in practice as hundreds of repeats of a short phrase with no real
// content, the kind of failure this whole fallback path exists to survive.
func sanitizeFallbackText(text string) []string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) > maxFallbackChars {
		trimmed = trimmed[:maxFallbackChars]
		if idx := strings.LastIndex(trimmed, " "); idx > maxFallbackChars/2 {
			trimmed = trimmed[:idx]
		}
		trimmed += "…"
	}

	var lines []string
	if strings.Contains(trimmed, "\n") {
		for _, l := range strings.Split(trimmed, "\n") {
			if l = strings.TrimSpace(l); l != "" {
				lines = append(lines, l)
			}
		}
	} else {
		for _, s := range strings.Split(trimmed, ". ") {
			if s = strings.TrimSpace(s); s != "" {
				lines = append(lines, s)
			}
		}
	}

	if len(lines) == 0 {
		return []string{"Sorry, I couldn't put together a clear answer for that - try rephrasing?"}
	}
	return lines
}

// normalizeBoard defends against a model putting the actual explanation into
// "title" instead of "lines" (observed in practice despite the schema/prompt
// both steering it toward "lines") - without this, a "lines" board with no
// lines renders as an empty board with nothing to type out. It returns false
// for a board that's unrecoverable (a "diagram" with no real mermaid content
// - observed in practice as keyword-salad dumped into "title" instead), which
// the caller drops rather than showing broken/garbage content.
func normalizeBoard(b *models.BoardContent) bool {
	if b.Kind == "lines" && len(b.Lines) == 0 && b.Title != "" {
		b.Lines = []string{b.Title}
		b.Title = ""
	}
	if b.Kind == "diagram" && b.Mermaid == "" {
		return false
	}
	return true
}
