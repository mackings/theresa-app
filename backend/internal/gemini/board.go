package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"google.golang.org/genai"

	"theresa/backend/internal/models"
)

// maxGenerateBoardWait bounds a single generation call. Gemini occasionally
// gets stuck in a degenerate repetition loop - observed in practice, a
// meta-question ("what is this built on?") produced four minutes of repeated
// boilerplate before finally stopping on its own. Without a ceiling, a single
// pathological generation leaves the user staring at "Theresa is working on
// it" for minutes with no way to tell it apart from a slow-but-healthy one;
// cutting it off falls through to the same fallback path a malformed
// response already takes.
const maxGenerateBoardWait = 45 * time.Second

const systemInstruction = `You are a patient, clear tutor. Teach the user's question or
document step by step, breaking the explanation into a sequence of "boards" suitable for
a visual teaching board. Use standard English only - no slang, no persona, no regional
dialect.

If asked about yourself - what you're built on, how you work, whether you're an AI - give
one short, honest board: you're an AI tutor that teaches step by step on a live board. Do
not invent specific technical details you don't actually know, and do not pad the answer
with restating the question or generic filler about your purpose. One or two plain
sentences is enough; then wait for what the student actually wants to learn.

IMPORTANT - pace yourself, don't dump everything at once: even when teaching from a large
uploaded document, cover at most 3-5 boards of real material, then STOP by emitting one
final "chat" board (see below) instead of continuing straight through the entire source.
The student needs a chance to actually respond before you keep going - assume there's more
material to cover on a later turn, don't try to fit a whole course into one response.

Respond with ONLY a JSON array of boards, shown to the student one after another. Each
board is an object with:
- "kind": "lines" (typed prose/math/code), "diagram" (a Mermaid diagram), or "chat" (a real
  conversational message shown in the chat panel, not written to the board)
- "title": a short optional heading for this board - a few words at most, never a full
  sentence or the explanation itself. The explanation always belongs in "lines". Not used
  for kind "chat".
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
- "message": (for kind "chat") REQUIRED, a short, genuinely conversational line - a real
  question or check-in, never a summary of what you just taught ("we covered X, Y, Z" is
  not engaging, it's a recap). Ask something that invites an actual reply: whether they want
  you to keep going, whether a specific part made sense, or what they'd like to focus on
  next. End your response with exactly one "chat" board whenever you pause - never end on a
  "lines"/"diagram" board with nothing inviting the student to respond.

Do not include any text outside the JSON array.`

// boardUnitSchema describes one board (an object discriminated by "kind"),
// used both as the array item schema for M3's one-shot response below and
// conceptually mirrored (as two separate per-tool schemas) by M4's live
// show_working/draw_diagram tools.
var boardUnitSchema = &genai.Schema{
	Type:     genai.TypeObject,
	Required: []string{"kind"},
	Properties: map[string]*genai.Schema{
		"kind":    {Type: genai.TypeString, Enum: []string{"lines", "diagram", "chat"}},
		"title":   {Type: genai.TypeString},
		"lines":   {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
		"mermaid": {Type: genai.TypeString},
		"message": {Type: genai.TypeString},
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
	// History is every prior turn of this session, oldest first - see
	// HistoryFromEvents. Without this, each call was a completely isolated,
	// context-free request: Gemini had no idea what had already been taught
	// or asked, which is why a vague follow-up ("yes, another example",
	// "can you show me another one") would land on an unrelated topic, and
	// why the student's chat replies never actually steered anything - the
	// model was answering each message in a vacuum, not having a
	// conversation.
	History []*genai.Content
}

// GenerateBoardStream asks Gemini to teach req.Text (optionally grounded in
// an uploaded file and/or prior conversation history), calling onBoard for
// each board as soon as Gemini finishes generating it - instead of
// buffering the entire multi-step response before returning anything. This
// is the exact same prompt/schema/model call as a one-shot request would
// use; only how the response is consumed changes (decoded element-by-element
// as it streams in, via the standard json.Decoder array-streaming pattern
// over a pipe fed by GenerateContentStream's chunks), so teaching
// content/order is unaffected - only how soon each step becomes visible.
//
// If Gemini's response isn't valid JSON at all, the full accumulated text is
// wrapped as a single lines-board and passed to onBoard, same fallback
// behavior as the previous one-shot implementation.
func (c *Client) GenerateBoardStream(ctx context.Context, req BoardRequest, onBoard func(models.BoardContent) error) error {
	parts := []*genai.Part{{Text: req.Text}}
	if req.FileURI != "" {
		parts = append(parts, &genai.Part{FileData: &genai.FileData{FileURI: req.FileURI, MIMEType: req.FileMimeType}})
	}
	contents := append(append([]*genai.Content{}, req.History...), genai.NewContentFromParts(parts, genai.RoleUser))

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, genai.RoleUser),
		ResponseMIMEType:  "application/json",
		ResponseSchema:    boardResponseSchema,
	}

	genCtx, cancel := context.WithTimeout(ctx, maxGenerateBoardWait)
	defer cancel()

	pr, pw := io.Pipe()
	var fullText strings.Builder
	var streamErr error

	go func() {
		for resp, err := range c.genai.Models.GenerateContentStream(genCtx, c.textModel, contents, config) {
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

	// The response never got further than raw JSON syntax - a truncated
	// array before even one field closed, or a repetition loop cut off
	// mid-string by the generation timeout. What's left is JSON noise, not
	// prose worth showing verbatim (observed in practice: literal "[", "{",
	// "\"kind\": \"lines\"," rendered as board text).
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		return []string{"Sorry, I had trouble putting together an answer for that - try asking again or rephrasing."}
	}

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
// lines renders as an empty board with nothing to type out. The title is run
// through the same sanitizeFallbackText used for invalid-JSON responses
// rather than moved over verbatim: a title is supposed to be "a few words at
// most" per the system prompt, but a degenerate/repetitive model response
// can still be syntactically valid JSON while dumping several thousand
// characters of repeated boilerplate into "title" (observed in practice -
// board.go's own generation timeout exists for the same underlying failure
// mode when it breaks the JSON instead) - sanitizeFallbackText's length cap
// and line-splitting bounds that same runaway text here too.
// normalizeBoard returns false for a board that's unrecoverable (a "diagram"
// with no real mermaid content - observed in practice as keyword-salad
// dumped into "title" instead), which the caller drops rather than showing
// broken/garbage content.
func normalizeBoard(b *models.BoardContent) bool {
	if b.Kind == "lines" && len(b.Lines) == 0 && b.Title != "" {
		b.Lines = sanitizeFallbackText(b.Title)
		b.Title = ""
	}
	if b.Kind == "diagram" && b.Mermaid == "" {
		return false
	}
	if b.Kind == "chat" && strings.TrimSpace(b.Message) == "" {
		return false
	}
	return true
}

// maxHistoryEvents bounds how many prior events get fed back as context -
// unbounded growth would make every later message in a long session slower
// and more expensive, and a Gemini call has its own context-window limits
// regardless. Recent history matters far more than a session's very first
// exchange for keeping a follow-up coherent.
const maxHistoryEvents = 30

// HistoryFromEvents converts a session's persisted event log into the
// alternating user/model turns BoardRequest.History needs for genuine
// conversational continuity. Without this, every call to GenerateBoardStream
// was a completely isolated, context-free request - see BoardRequest's doc
// comment. board_update and chat_message (both always the assistant's own
// output) map to Gemini's "model" role; user_text maps to "user". A
// board_update's content is reconstructed as plain readable text (title +
// lines, or a mermaid block) rather than re-serializing the original JSON
// shape - Gemini only needs to know what was already covered, not the
// literal wire format of its own prior output.
func HistoryFromEvents(events []models.SessionEvent) []*genai.Content {
	if len(events) > maxHistoryEvents {
		events = events[len(events)-maxHistoryEvents:]
	}

	var history []*genai.Content
	for _, e := range events {
		switch e.Type {
		case "user_text":
			if e.Text == "" {
				continue
			}
			history = append(history, genai.NewContentFromText(e.Text, genai.RoleUser))
		case "chat_message":
			if e.Text == "" {
				continue
			}
			history = append(history, genai.NewContentFromText(e.Text, genai.RoleModel))
		case "board_update":
			if e.Board == nil {
				continue
			}
			if text := boardContentAsText(*e.Board); text != "" {
				history = append(history, genai.NewContentFromText(text, genai.RoleModel))
			}
		}
	}
	return history
}

func boardContentAsText(b models.BoardContent) string {
	var sb strings.Builder
	if b.Title != "" {
		sb.WriteString(b.Title)
		sb.WriteString("\n")
	}
	if b.Kind == "diagram" {
		sb.WriteString(b.Mermaid)
		return sb.String()
	}
	for i, line := range b.Lines {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(line)
	}
	return sb.String()
}
