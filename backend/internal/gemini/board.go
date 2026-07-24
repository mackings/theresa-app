package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// GenerateBoardStream asks Gemini to teach req.Text (optionally grounded in
// an uploaded file), calling onBoard for each board as soon as Gemini
// finishes generating it - instead of buffering the entire multi-step
// response before returning anything. This is the exact same prompt/schema/
// model call as a one-shot request would use; only how the response is
// consumed changes (decoded element-by-element as it streams in, via the
// standard json.Decoder array-streaming pattern over a pipe fed by
// GenerateContentStream's chunks), so teaching content/order is unaffected -
// only how soon each step becomes visible.
//
// If Gemini's response isn't valid JSON at all, the full accumulated text is
// wrapped as a single lines-board and passed to onBoard, same fallback
// behavior as the previous one-shot implementation.
func (c *Client) GenerateBoardStream(ctx context.Context, req BoardRequest, onBoard func(models.BoardContent) error) error {
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

	pr, pw := io.Pipe()
	var fullText strings.Builder
	var streamErr error

	go func() {
		for resp, err := range c.genai.Models.GenerateContentStream(ctx, c.textModel, []*genai.Content{content}, config) {
			if err != nil {
				streamErr = err
				pw.CloseWithError(err)
				return
			}
			chunk := resp.Text()
			fullText.WriteString(chunk)
			if _, werr := pw.Write([]byte(chunk)); werr != nil {
				return
			}
		}
		pw.Close()
	}()

	decoder := json.NewDecoder(pr)
	opened := false
	if tok, err := decoder.Token(); err == nil {
		delim, ok := tok.(json.Delim)
		opened = ok && delim == '['
	}

	any := false
	if opened {
		for decoder.More() {
			var b models.BoardContent
			if err := decoder.Decode(&b); err != nil {
				break
			}
			if normalizeBoard(&b) {
				any = true
				if err := onBoard(b); err != nil {
					io.Copy(io.Discard, pr) //nolint:errcheck // best-effort drain to release the producer goroutine
					return err
				}
			}
		}
	}

	// Drain any remaining stream output so the producer goroutine's next
	// write never blocks forever on a reader nobody's servicing anymore
	// (e.g. we stopped early because of a malformed array element) - and,
	// for the invalid-JSON fallback path below, so fullText is guaranteed
	// complete by the time we read it (the goroutine appends to fullText
	// before writing each chunk to the pipe, and only closes pw as its very
	// last action, so draining to a closed pipe means fullText is done too).
	io.Copy(io.Discard, pr) //nolint:errcheck

	if any {
		return nil
	}
	if streamErr != nil {
		return fmt.Errorf("generate board: %w", streamErr)
	}
	return onBoard(models.BoardContent{Kind: "lines", Lines: sanitizeFallbackText(fullText.String())})
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
