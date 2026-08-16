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

// ShowCodeFunctionDeclaration lets the model show a real, syntax-highlighted,
// multi-line code example on its own board - for a programming lesson's
// actual code worth studying, not a short inline mention (show_working's
// existing inline-backtick/fenced-block support within "lines" already
// covers that shorter case). Call this alongside show_working for narration,
// never as a substitute for actually explaining what the code does.
var ShowCodeFunctionDeclaration = &genai.FunctionDeclaration{
	Name:        "show_code",
	Description: "Show a real, syntax-highlighted, multi-line code example on its own board - for a genuine teaching example, not a short inline mention (use show_working's inline backticks/fenced block for that).",
	Parameters: &genai.Schema{
		Type:     genai.TypeObject,
		Required: []string{"code", "language"},
		Properties: map[string]*genai.Schema{
			"title":    {Type: genai.TypeString},
			"code":     {Type: genai.TypeString},
			"language": {Type: genai.TypeString},
		},
	},
}

// scene3DShapeEnum is the closed set of primitive shapes the procedural 3D
// scene vocabulary can use - matches the Three.js geometries ThreeDBoard.tsx
// actually renders (sphereGeometry/boxGeometry/etc.), never arbitrary
// shape names.
var scene3DShapeEnum = []string{"sphere", "box", "cylinder", "cone", "torus", "capsule"}

// Show3DModelFunctionDeclaration lets the model show a small interactive 3D
// scene - either a real, curated anatomy model (asset_key, a closed enum -
// see AnatomyAssets) or a small procedural scene of labeled positioned
// shapes (parts, optionally connected by links) for anything else: a
// molecule, a geometric shape, a kid-friendly diagram, or an anatomy topic
// not yet in the curated real-asset set. Exactly one of asset_key/parts is
// meaningful per call - BuildBoardContent decides which, preferring
// asset_key when it's a recognized key (see its own doc comment).
var Show3DModelFunctionDeclaration = &genai.FunctionDeclaration{
	Name: "show_3d_model",
	Description: "Show a small interactive 3D scene the student can drag to rotate. For one of the " +
		"curated real anatomy organs, pass asset_key. For anything else - a molecule, a geometric " +
		"shape, a kid-friendly diagram, or an anatomy topic not in the curated list - pass parts " +
		"(and optionally links) instead: simple labeled shapes positioned in 3D space.",
	Parameters: &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"caption":   {Type: genai.TypeString},
			"asset_key": {Type: genai.TypeString, Format: "enum", Enum: AnatomyAssetKeys()},
			"parts": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type:     genai.TypeObject,
					Required: []string{"label", "shape", "x", "y", "z"},
					Properties: map[string]*genai.Schema{
						"label": {Type: genai.TypeString},
						"shape": {Type: genai.TypeString, Format: "enum", Enum: scene3DShapeEnum},
						"color": {Type: genai.TypeString},
						"size":  {Type: genai.TypeNumber},
						"x":     {Type: genai.TypeNumber},
						"y":     {Type: genai.TypeNumber},
						"z":     {Type: genai.TypeNumber},
					},
				},
			},
			"links": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type:     genai.TypeObject,
					Required: []string{"from", "to"},
					Properties: map[string]*genai.Schema{
						"from":  {Type: genai.TypeString},
						"to":    {Type: genai.TypeString},
						"label": {Type: genai.TypeString},
					},
				},
			},
		},
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
		ShowCodeFunctionDeclaration,
		Show3DModelFunctionDeclaration,
		ClearBoardFunctionDeclaration,
	}},
}

// TextTools is what text mode registers - BoardTools plus chat_checkin,
// since text has no other channel for a genuine conversational pause.
var TextTools = []*genai.Tool{
	{FunctionDeclarations: []*genai.FunctionDeclaration{
		ShowWorkingFunctionDeclaration,
		DrawDiagramFunctionDeclaration,
		ShowCodeFunctionDeclaration,
		Show3DModelFunctionDeclaration,
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

	case "show_code":
		code, _ := args["code"].(string)
		if strings.TrimSpace(code) == "" {
			return models.BoardContent{}, false
		}
		language, _ := args["language"].(string)
		title, _ := args["title"].(string)
		return models.BoardContent{Kind: "code", Title: title, Code: code, CodeLanguage: normalizeCodeLanguage(language)}, true

	case "show_3d_model":
		caption, _ := args["caption"].(string)
		if assetKey, _ := args["asset_key"].(string); assetKey != "" {
			if _, ok := AnatomyAssets[assetKey]; ok {
				return models.BoardContent{Kind: "3d", Scene3D: &models.Scene3D{AssetKey: assetKey, Caption: caption}}, true
			}
			// An unrecognized/hallucinated asset_key isn't a hard failure -
			// fall through to the procedural parts/links path below, same as
			// if asset_key had never been given at all.
		}
		parts, ok := parseScene3DParts(args["parts"])
		if !ok {
			return models.BoardContent{}, false
		}
		links := parseScene3DLinks(args["links"], parts)
		return models.BoardContent{Kind: "3d", Scene3D: &models.Scene3D{Caption: caption, Parts: parts, Links: links}}, true

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

// maxScene3DParts/maxScene3DLinks bound a procedural 3D scene's size -
// same "Gemini schema compliance alone isn't reliable, cap in code" spirit
// as every other defensive limit in this file, since this schema (like
// learningplan.go's) deliberately carries no MinItems/MaxItems.
const maxScene3DParts = 12
const maxScene3DLinks = 20

var validScene3DShapes = map[string]bool{
	"sphere": true, "box": true, "cylinder": true, "cone": true, "torus": true, "capsule": true,
}

// parseScene3DParts reads the "parts" argument off a show_3d_model call -
// manual type assertions per element (never a blind struct unmarshal, same
// discipline as the rest of this file), tolerant of individual bad
// elements rather than failing the whole call over one. An element missing
// a non-empty label is dropped outright; an unrecognized (or missing)
// shape defaults to "sphere" rather than dropping the part, since the part
// still needs to exist for any link pointing at it to remain valid. Size
// defaults to 1 when absent or non-positive. Returns ok=false only when
// zero usable parts remain.
func parseScene3DParts(raw any) ([]models.Scene3DPart, bool) {
	items, ok := raw.([]any)
	if !ok {
		return nil, false
	}

	parts := make([]models.Scene3DPart, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		label, _ := m["label"].(string)
		if strings.TrimSpace(label) == "" {
			continue
		}
		shape, _ := m["shape"].(string)
		if !validScene3DShapes[shape] {
			shape = "sphere"
		}
		size, _ := m["size"].(float64)
		if size <= 0 {
			size = 1
		}
		x, _ := m["x"].(float64)
		y, _ := m["y"].(float64)
		z, _ := m["z"].(float64)
		color, _ := m["color"].(string)
		parts = append(parts, models.Scene3DPart{
			Label: label, Shape: shape, Color: color, Size: size, X: x, Y: y, Z: z,
		})
		if len(parts) >= maxScene3DParts {
			break
		}
	}

	return parts, len(parts) > 0
}

// parseScene3DLinks reads the "links" argument, dropping any link whose
// From/To doesn't match one of parts' labels - never trusts a dangling
// reference to the frontend renderer. An empty/absent/malformed links
// argument is fine (most scenes have none); this never fails the call.
func parseScene3DLinks(raw any, parts []models.Scene3DPart) []models.Scene3DLink {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}

	validLabels := make(map[string]bool, len(parts))
	for _, p := range parts {
		validLabels[p.Label] = true
	}

	links := make([]models.Scene3DLink, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		from, _ := m["from"].(string)
		to, _ := m["to"].(string)
		if !validLabels[from] || !validLabels[to] {
			continue
		}
		label, _ := m["label"].(string)
		links = append(links, models.Scene3DLink{From: from, To: to, Label: label})
		if len(links) >= maxScene3DLinks {
			break
		}
	}

	return links
}

// codeLanguageAliases maps common shorthand/alternate spellings Gemini might
// use to the exact language keys the frontend's CodeMirror language-
// extension map looks up by - an unrecognized language isn't an error case,
// it just falls through to the raw lowercased value, which the frontend
// treats as plain (unhighlighted) text rather than crashing.
var codeLanguageAliases = map[string]string{
	"py":     "python",
	"js":     "javascript",
	"jsx":    "javascript",
	"ts":     "typescript",
	"tsx":    "typescript",
	"golang": "go",
	"c++":    "cpp",
	"c#":     "csharp",
	"shell":  "bash",
	"sh":     "bash",
	"html5":  "html",
}

// normalizeCodeLanguage lowercases and resolves common aliases so the
// frontend's language-extension lookup doesn't need to duplicate this
// mapping - see codeLanguageAliases.
func normalizeCodeLanguage(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if alias, ok := codeLanguageAliases[lower]; ok {
		return alias
	}
	return lower
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
	scene3DKey := ""
	if b.Scene3D != nil {
		scene3DKey = b.Scene3D.AssetKey
		for _, p := range b.Scene3D.Parts {
			scene3DKey += p.Label
		}
	}
	joined := strings.Join(b.Lines, " ") + "|" + b.Mermaid + "|" + b.Code + "|" + b.Message + "|" + scene3DKey
	normalized := strings.Join(strings.Fields(joined), " ")
	normalized = strings.TrimSuffix(normalized, "…")
	if len(normalized) > boardKeyPrefixLen {
		normalized = normalized[:boardKeyPrefixLen]
	}
	return b.Kind + "|" + normalized
}
