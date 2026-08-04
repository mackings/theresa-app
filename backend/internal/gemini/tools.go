package gemini

import (
	"encoding/json"
	"strings"

	"google.golang.org/genai"

	"theresa/backend/internal/models"
)

// ShowWorkingFunctionDeclaration lets the model show a whole board's worth of
// typed working - prose, math, and/or code lines - one board at a time,
// instead of planning an entire multi-step lesson upfront as one large
// structured response. Shared by text mode (board.go's GenerateBoardStream)
// and voice (gemini/live) - both call the exact same function, which is what
// keeps text mode's per-step latency as fast as voice's already was, instead
// of asking Gemini to plan/validate a whole lesson before anything renders.
var ShowWorkingFunctionDeclaration = &genai.FunctionDeclaration{
	Name:        "show_working",
	Description: "Show a board's worth of typed working - prose, math, and/or code lines - one board at a time, in sync with what you're saying or writing.",
	Parameters: &genai.Schema{
		Type:     genai.TypeObject,
		Required: []string{"lines"},
		Properties: map[string]*genai.Schema{
			"title": {Type: genai.TypeString},
			"lines": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
		},
	},
}

// DrawDiagramFunctionDeclaration lets the model draw a Mermaid diagram - only
// for cycles, branches, or sequences, never a numeric graph/plot (Mermaid
// can't render axes or plotted data meaningfully).
var DrawDiagramFunctionDeclaration = &genai.FunctionDeclaration{
	Name:        "draw_diagram",
	Description: "Draw a Mermaid diagram for a cycle, branch, or sequence - never for a numeric graph/plot.",
	Parameters: &genai.Schema{
		Type:     genai.TypeObject,
		Required: []string{"mermaid"},
		Properties: map[string]*genai.Schema{
			"title":   {Type: genai.TypeString},
			"mermaid": {Type: genai.TypeString},
		},
	},
}

// ClearBoardFunctionDeclaration lets the model actually erase the board when
// the student explicitly asks for it ("clean the board", "clear this",
// "start fresh") - without it, the board has no way to become blank again on
// request, since every other tool only ever adds a new board on top of the
// last one. No parameters - clearing has no content of its own.
var ClearBoardFunctionDeclaration = &genai.FunctionDeclaration{
	Name:        "clear_board",
	Description: "Erase the board back to blank - call this only when the student explicitly asks to clear, clean, or erase the board.",
	Parameters: &genai.Schema{
		Type:       genai.TypeObject,
		Properties: map[string]*genai.Schema{},
	},
}

// ChatCheckInFunctionDeclaration is text mode's equivalent of voice simply
// pausing to ask a real question out loud. Text has no speech channel, so a
// genuine conversational check-in needs its own explicit call to be told
// apart from board content and rendered as a real chat bubble (see
// httpapi.PostMessage) instead of typed onto the board. Voice doesn't
// register this tool - it just talks (see live.PersonaInstruction's own
// pacing instructions), so this is text-only (see TextTools below).
var ChatCheckInFunctionDeclaration = &genai.FunctionDeclaration{
	Name:        "chat_checkin",
	Description: "Pause to ask the student a real, genuine question - never a recap of what was just covered.",
	Parameters: &genai.Schema{
		Type:     genai.TypeObject,
		Required: []string{"message"},
		Properties: map[string]*genai.Schema{
			"message": {Type: genai.TypeString},
		},
	},
}

// BoardTools is what voice registers - show_working/draw_diagram only, since
// voice paces itself by simply talking, not by calling a function.
var BoardTools = []*genai.Tool{
	{FunctionDeclarations: []*genai.FunctionDeclaration{
		ShowWorkingFunctionDeclaration,
		DrawDiagramFunctionDeclaration,
		ClearBoardFunctionDeclaration,
	}},
}

// TextTools is what text mode registers - BoardTools plus chat_checkin,
// since text has no other channel for a genuine conversational pause.
var TextTools = []*genai.Tool{
	{FunctionDeclarations: []*genai.FunctionDeclaration{
		ShowWorkingFunctionDeclaration,
		DrawDiagramFunctionDeclaration,
		ChatCheckInFunctionDeclaration,
		ClearBoardFunctionDeclaration,
	}},
}

// BuildBoardContent turns a function call into a models.BoardContent. It
// reads fields directly off the args map (rather than round-tripping through
// json.Marshal/Unmarshal into a struct, which would silently discard a
// type-mismatch error on a field like "lines") and returns ok=false for
// anything that doesn't yield real, usable content - shared by voice's
// handleToolCall and text mode's GenerateBoardStream, both of which respond
// to a false ok with a rejecting function-response rather than silently
// treating a malformed or empty call as having succeeded.
func BuildBoardContent(name string, args map[string]any) (models.BoardContent, bool) {
	switch name {
	case "show_working":
		lines, ok := CoerceStringArray(args["lines"])
		if !ok || len(lines) == 0 {
			return models.BoardContent{}, false
		}
		title, _ := args["title"].(string)
		return models.BoardContent{Kind: "lines", Title: title, Lines: RepairOrphanedListMarkers(RepairOverFractions(lines))}, true

	case "draw_diagram":
		mermaid, _ := args["mermaid"].(string)
		if strings.TrimSpace(mermaid) == "" {
			return models.BoardContent{}, false
		}
		title, _ := args["title"].(string)
		return models.BoardContent{Kind: "diagram", Title: title, Mermaid: mermaid}, true

	case "chat_checkin":
		message, _ := args["message"].(string)
		if strings.TrimSpace(message) == "" {
			return models.BoardContent{}, false
		}
		return models.BoardContent{Kind: "chat", Message: message}, true

	case "clear_board":
		return models.BoardContent{Kind: "clear"}, true

	default:
		return models.BoardContent{}, false
	}
}

// CoerceStringArray handles Gemini's preview-model quirk of occasionally
// sending an array-typed argument JSON-encoded as a string instead of a real
// array.
func CoerceStringArray(raw any) ([]string, bool) {
	switch v := raw.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out, len(out) > 0
	case string:
		var arr []string
		if err := json.Unmarshal([]byte(v), &arr); err != nil {
			return nil, false
		}
		return arr, len(arr) > 0
	default:
		return nil, false
	}
}

// boardKeyPrefixLen bounds how much of a board's normalized content the
// duplicate check compares - see BoardKey. Long enough to be a genuine
// content fingerprint, short enough that a repeat which got cut off partway
// through (observed in practice, ending in "…") still matches the original.
const boardKeyPrefixLen = 80

// BoardKey fingerprints a board's actual content (not title, which the model
// sometimes varies slightly even on an otherwise-identical repeat) for
// duplicate-repetition detection - shared by voice's handleToolCall and text
// mode's GenerateBoardStream. Both guard against the same observed failure
// class: nothing inherently stops the model from calling show_working (or,
// in text mode, requesting another turn) with essentially the same content
// it just produced. Deliberately fuzzy, not byte-exact: a real observed
// repeat came back with different whitespace on wrapped bullet lines and
// truncated partway through rather than as a perfect duplicate, so this
// collapses whitespace and compares only a fixed-length prefix rather than
// the full, exact content.
func BoardKey(b models.BoardContent) string {
	joined := strings.Join(b.Lines, " ") + "|" + b.Mermaid + "|" + b.Message
	normalized := strings.Join(strings.Fields(joined), " ")
	normalized = strings.TrimSuffix(normalized, "…")
	if len(normalized) > boardKeyPrefixLen {
		normalized = normalized[:boardKeyPrefixLen]
	}
	return b.Kind + "|" + normalized
}
