package gemini

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"google.golang.org/genai"

	"theresa/backend/internal/models"
)

// maxBoardsPerTurn bounds how many function calls one PostMessage turn will
// make before stopping regardless of what the model wants to do next - a
// safety net against a runaway loop that never calls chat_checkin, mirroring
// voice's own implicit pacing (persona.go's "after roughly 2-3 show_working
// calls, stop and ask a real question").
const maxBoardsPerTurn = 8

// maxSingleCallWait bounds one GenerateContent call - now asking for a
// single board or a short reply, not an entire multi-step lesson planned and
// validated as one large structured response, so it needs nowhere near the
// old one-shot budget. A stall here costs at most this long, not the whole
// turn - see the "any" check in GenerateBoardStream below.
const maxSingleCallWait = 20 * time.Second

const incrementalSystemInstruction = `You are a patient, clear tutor. Teach the user's question
or document step by step. Respond in whichever language the student is writing in - if they
write in Yoruba, Igbo, Hausa, or another language, teach in that same language, not English.
Default to standard English only when the student hasn't indicated a language preference. No
slang, no persona, no regional dialect - within whichever language you're using, keep it
standard and clear.

If asked about yourself - what you're built on, how you work, whether you're an AI - give one
short, honest answer via show_working: you're an AI tutor that teaches step by step on a live
board. Do not invent specific technical details you don't actually know, and do not pad the
answer with restating the question or generic filler about your purpose. One or two plain
sentences is enough; then call chat_checkin to wait for what the student actually wants to
learn.

You have four tools:
- show_working(title?, lines): show a board's worth of typed working - prose, math, and/or
  code. Call it once per board (you can call it again later for the next board), not once per
  line. A line may contain inline math wrapped in single dollar signs ($...$), inline code
  wrapped in backticks (` + "`...`" + `), or be an entire fenced code block. Keep each board's lines
  focused on one idea - not a wall of text. Every math expression must appear exactly once,
  fully resolved - never repeat a partially-worked step across multiple lines. Never merge a
  heading-like phrase into the start of a line - use "title" for that instead - and never run
  two unrelated sentences together with no natural break between them. For a numbered or
  bulleted list, each item's marker and its text belong together on the SAME line, and a bare
  marker must never appear as its own separate line - wrong: ["We look for two numbers that:
  1", "Multiply to give c", "2", "Add up to give b"], right: ["We look for two numbers that:",
  "1. Multiply to give c", "2. Add up to give b"]. For a fraction, always use
  \frac{numerator}{denominator} - never the older \over syntax (e.g. "$a \over b$"), which
  renders incorrectly in this app's math renderer. Any multi-letter abbreviation or
  descriptive word used inside math mode ($...$) must be wrapped in \text{...} - write
  $\text{PED} = \frac{\text{Change in Quantity}}{\text{Change in Price}}$, never bare
  $PED = ...$: a bare multi-letter token in math mode renders as separate spaced-out
  italic letters (e.g. "P E D"), not as a normal word or abbreviation.
- draw_diagram(title?, mermaid): draw a Mermaid diagram - for a genuine cycle, branch, or
  sequence of steps, never a numeric graph or plot (Mermaid can't render axes or plotted data -
  describe that in words via show_working instead). Prefer calling this in addition to
  show_working, even when not explicitly asked for a diagram or picture, whenever what you're
  explaining has a natural shape a diagram can show - a visual often makes that kind of
  structure clearer than text alone. Skip it for content with no natural diagram shape (a
  plain definition, a formula, a numeric fact), where show_working alone is the right call.
- chat_checkin(message): pause to ask the student a real, genuine question - never a summary of
  what you just covered ("we covered X, Y, Z" is not engaging, it's a recap). Ask something
  that invites an actual reply: whether they want you to keep going, whether a specific part
  made sense, or what they'd like to focus on next.
- clear_board(): erase the board back to blank. Call this ONLY when the student explicitly
  asks you to clear, clean, or erase the board (or start fresh/wipe it) - never on your own
  initiative, and never as a substitute for show_working when starting a new topic (a new
  topic just gets its own show_working call; the board naturally keeps growing with each new
  board underneath the last, which is the intended behavior).

IMPORTANT - pace yourself, don't dump everything at once: even when teaching from a large
uploaded document, cover at most 3-5 boards of real material, then call chat_checkin instead
of continuing straight through the entire source. The student needs a chance to actually
respond before you keep going - assume there's more material to cover on a later turn. Always
call chat_checkin when you pause - never stop mid-explanation with nothing inviting the
student to respond.`

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
// an uploaded file and/or prior conversation history) via function calling -
// one show_working/draw_diagram call per board, exactly the same mechanism
// voice already used successfully - instead of the old approach of asking
// for an entire multi-step lesson as one large structured JSON array in a
// single call. That approach required the model to effectively plan and
// validate the whole response before anything could render, which was both
// slower (nothing shows until the entire structure is ready, unlike a single
// small function call) and less reliable (a big structured-output call is
// more prone to stalling/looping - the exact "context deadline exceeded"
// failures chased throughout this session's testing). onBoard is called
// once per successful function call, in order, same external contract as
// before so PostMessage's caller-side logic didn't need to change.
//
// Each individual call gets its own maxSingleCallWait budget. If a call
// fails after at least one board was already shown this turn, that's
// treated as a graceful stopping point (return nil) rather than a hard
// failure - a partial, real answer is better than discarding it to retry
// the whole turn. Only an error before anything was shown propagates up,
// matching the invariant PostMessage's retry-once logic depends on.
func (c *Client) GenerateBoardStream(ctx context.Context, req BoardRequest, onBoard func(models.BoardContent) error) error {
	parts := []*genai.Part{{Text: req.Text}}
	if req.FileURI != "" {
		parts = append(parts, &genai.Part{FileData: &genai.FileData{FileURI: req.FileURI, MIMEType: req.FileMimeType}})
	}
	contents := append(append([]*genai.Content{}, req.History...), genai.NewContentFromParts(parts, genai.RoleUser))

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(incrementalSystemInstruction, genai.RoleUser),
		Tools:             TextTools,
	}

	producedAny := false
	lastKey := ""

	for i := 0; i < maxBoardsPerTurn; i++ {
		callCtx, cancel := context.WithTimeout(ctx, maxSingleCallWait)
		resp, err := c.genai.Models.GenerateContent(callCtx, c.textModel, contents, config)
		cancel()
		if err != nil {
			if producedAny {
				return nil
			}
			return fmt.Errorf("generate board: %w", err)
		}

		calls := resp.FunctionCalls()
		if len(calls) == 0 {
			// No function call - the model responded with plain text instead
			// (or nothing usable). Treat any text as a fallback board and
			// stop; this is also how a turn naturally ends if the model has
			// nothing more to add without calling chat_checkin.
			if text := strings.TrimSpace(resp.Text()); text != "" {
				if err := onBoard(models.BoardContent{Kind: "lines", Lines: sanitizeFallbackText(text)}); err != nil {
					return err
				}
				producedAny = true
			}
			break
		}

		stop := false
		for _, call := range calls {
			board, ok := BuildBoardContent(call.Name, call.Args)

			var response map[string]any
			switch {
			case !ok:
				response = map[string]any{"error": fmt.Sprintf(
					"%s call had empty or malformed arguments; retry with real content", call.Name)}

			case BoardKey(board) == lastKey:
				response = map[string]any{"error": fmt.Sprintf(
					"%s call repeated the exact same content as your last call - move on to something new, or call chat_checkin to pause instead", call.Name)}

			default:
				response = map[string]any{"output": "ok"}
				lastKey = BoardKey(board)
				if err := onBoard(board); err != nil {
					return err
				}
				producedAny = true
				if call.Name == "chat_checkin" {
					stop = true
				}
			}

			contents = append(contents,
				genai.NewContentFromFunctionCall(call.Name, call.Args, genai.RoleModel),
				genai.NewContentFromFunctionResponse(call.Name, response, genai.RoleUser),
			)
		}
		if stop {
			break
		}
	}

	if !producedAny {
		return errors.New("generate board: no usable content produced")
	}
	return nil
}

const maxFallbackChars = 600

// sanitizeFallbackText turns raw non-function-call model output into a few
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

var bareListMarkerRe = regexp.MustCompile(`^(\d{1,2}|-)$`)

// RepairOrphanedListMarkers defends against an observed Gemini quirk where a
// numbered/bulleted list's marker occasionally ends up as its own bare
// array element ("2" with nothing else) instead of prefixed onto its item's
// text, despite the system prompt's explicit instruction and example
// against it (prompt compliance alone isn't reliable - observed in practice
// across repeated identical requests, sometimes clean, sometimes not).
// Only handles the unambiguous case (a line that, trimmed, is nothing but
// the marker itself) - a marker merged onto the END of the PRECEDING line's
// text is a much more ambiguous pattern to detect safely (a line
// legitimately ending in a number, like a computed value, would look
// identical) and is deliberately left alone rather than risk mangling real
// content.
func RepairOrphanedListMarkers(lines []string) []string {
	repaired := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if bareListMarkerRe.MatchString(trimmed) && i+1 < len(lines) {
			marker := trimmed
			if marker != "-" {
				marker += "."
			}
			lines[i+1] = marker + " " + strings.TrimSpace(lines[i+1])
			continue // drop this bare-marker line - its content moved onto the next
		}
		repaired = append(repaired, lines[i])
	}
	return repaired
}

// overFractionRe matches a whole inline-math segment built with the older
// TeX \over fraction syntax (e.g. "$\text{A} \over \text{B}$"). Deliberately
// simple - a non-greedy match between the two $ delimiters, split on \over -
// rather than a brace-aware parser: the case actually observed has no
// nested $ signs, and if a genuinely unusual expression doesn't match
// cleanly, ReplaceAllStringFunc just leaves that segment untouched (a safe
// failure mode - no risk of mangling something this pattern wasn't meant
// to handle).
var overFractionRe = regexp.MustCompile(`\$([^$]*?)\\over([^$]*?)\$`)

// RepairOverFractions defends against an observed Gemini quirk: despite the
// system prompt's explicit instruction to use \frac{}{}, the model
// occasionally still writes a fraction with the older \over syntax instead
// (e.g. "$\text{Percentage Change in Demand} \over \text{Percentage Change
// in Price}$") - which this app's KaTeX renderer doesn't handle cleanly,
// observed in practice rendering as each character spelled out individually
// in math-italic with spaces between them rather than an actual fraction.
// Rewrites the whole segment as $\frac{numerator}{denominator}$, which KaTeX
// renders correctly. Prompt compliance alone isn't reliable (the same
// lesson already learned from RepairOrphanedListMarkers above), so this is
// the same defense-in-depth pattern applied to a different Gemini quirk.
func RepairOverFractions(lines []string) []string {
	repaired := make([]string, len(lines))
	for i, line := range lines {
		repaired[i] = overFractionRe.ReplaceAllStringFunc(line, func(match string) string {
			parts := overFractionRe.FindStringSubmatch(match)
			if len(parts) != 3 {
				return match
			}
			numerator := strings.TrimSpace(parts[1])
			denominator := strings.TrimSpace(parts[2])
			if numerator == "" || denominator == "" {
				return match
			}
			return fmt.Sprintf(`$\frac{%s}{%s}$`, numerator, denominator)
		})
	}
	return repaired
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
