package gemini

import (
	"testing"

	"theresa/backend/internal/models"
)

func TestCoerceStringArray(t *testing.T) {
	cases := []struct {
		name    string
		input   any
		wantOK  bool
		wantLen int
	}{
		{"real array", []any{"line one", "line two"}, true, 2},
		{"json-encoded string quirk", `["line one", "line two", "line three"]`, true, 3},
		{"empty array", []any{}, false, 0},
		{"empty string", "", false, 0},
		{"malformed json string", "not json", false, 0},
		{"wrong type", 42, false, 0},
		{"nil", nil, false, 0},
		{"array with empty strings filtered", []any{"", "real line", ""}, true, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines, ok := CoerceStringArray(tc.input)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v (lines=%v)", ok, tc.wantOK, lines)
			}
			if ok && len(lines) != tc.wantLen {
				t.Fatalf("len(lines) = %d, want %d (lines=%v)", len(lines), tc.wantLen, lines)
			}
		})
	}
}

func TestBuildBoardContent(t *testing.T) {
	t.Run("show_working with real lines", func(t *testing.T) {
		board, ok := BuildBoardContent("show_working", map[string]any{
			"title": "Step 1",
			"lines": []any{"The area is $A = \\pi r^2$"},
		})
		if !ok {
			t.Fatal("expected ok=true")
		}
		if board.Kind != "lines" || board.Title != "Step 1" || len(board.Lines) != 1 {
			t.Fatalf("unexpected board: %+v", board)
		}
	})

	t.Run("show_working with quirky string-encoded lines", func(t *testing.T) {
		board, ok := BuildBoardContent("show_working", map[string]any{
			"lines": `["first line", "second line"]`,
		})
		if !ok || len(board.Lines) != 2 {
			t.Fatalf("expected coerced lines, got ok=%v board=%+v", ok, board)
		}
	})

	t.Run("show_working with empty lines fails", func(t *testing.T) {
		_, ok := BuildBoardContent("show_working", map[string]any{"lines": []any{}})
		if ok {
			t.Fatal("expected ok=false for empty lines")
		}
	})

	t.Run("draw_diagram with real mermaid", func(t *testing.T) {
		board, ok := BuildBoardContent("draw_diagram", map[string]any{
			"mermaid": "graph TD; A-->B;",
		})
		if !ok || board.Kind != "diagram" || board.Mermaid == "" {
			t.Fatalf("unexpected result: ok=%v board=%+v", ok, board)
		}
	})

	t.Run("draw_diagram with blank mermaid fails", func(t *testing.T) {
		_, ok := BuildBoardContent("draw_diagram", map[string]any{"mermaid": "   "})
		if ok {
			t.Fatal("expected ok=false for blank mermaid")
		}
	})

	t.Run("chat_checkin with real message", func(t *testing.T) {
		board, ok := BuildBoardContent("chat_checkin", map[string]any{"message": "Does this make sense so far?"})
		if !ok || board.Kind != "chat" || board.Message == "" {
			t.Fatalf("unexpected result: ok=%v board=%+v", ok, board)
		}
	})

	t.Run("chat_checkin with blank message fails", func(t *testing.T) {
		_, ok := BuildBoardContent("chat_checkin", map[string]any{"message": "   "})
		if ok {
			t.Fatal("expected ok=false for blank message")
		}
	})

	t.Run("unknown tool name fails", func(t *testing.T) {
		_, ok := BuildBoardContent("update_board", map[string]any{"step": 1})
		if ok {
			t.Fatal("expected ok=false for unhandled tool name")
		}
	})

	t.Run("show_3d_model with a known asset_key uses the real asset path", func(t *testing.T) {
		board, ok := BuildBoardContent("show_3d_model", map[string]any{
			"caption":   "The liver",
			"asset_key": "liver",
		})
		if !ok || board.Kind != "3d" || board.Scene3D == nil {
			t.Fatalf("unexpected result: ok=%v board=%+v", ok, board)
		}
		if board.Scene3D.AssetKey != "liver" || len(board.Scene3D.Parts) != 0 {
			t.Fatalf("expected asset_key path with no procedural parts: %+v", board.Scene3D)
		}
	})

	t.Run("show_3d_model prefers a known asset_key even when parts are also present", func(t *testing.T) {
		board, ok := BuildBoardContent("show_3d_model", map[string]any{
			"asset_key": "kidneys",
			"parts":     []any{map[string]any{"label": "X", "shape": "sphere", "x": 0.0, "y": 0.0, "z": 0.0}},
		})
		if !ok || board.Scene3D.AssetKey != "kidneys" || len(board.Scene3D.Parts) != 0 {
			t.Fatalf("expected asset_key to take precedence over parts: ok=%v board=%+v", ok, board)
		}
	})

	t.Run("show_3d_model with an unrecognized asset_key falls back to procedural parts", func(t *testing.T) {
		board, ok := BuildBoardContent("show_3d_model", map[string]any{
			"asset_key": "appendix", // not a curated key
			"parts":     []any{map[string]any{"label": "Blob", "shape": "sphere", "x": 0.0, "y": 0.0, "z": 0.0}},
		})
		if !ok || board.Scene3D.AssetKey != "" || len(board.Scene3D.Parts) != 1 {
			t.Fatalf("expected fallback to procedural parts: ok=%v board=%+v", ok, board)
		}
	})

	t.Run("show_3d_model with real parts and no asset_key uses the procedural path", func(t *testing.T) {
		board, ok := BuildBoardContent("show_3d_model", map[string]any{
			"caption": "Water molecule",
			"parts": []any{
				map[string]any{"label": "O", "shape": "sphere", "color": "red", "x": 0.0, "y": 0.0, "z": 0.0},
				map[string]any{"label": "H1", "shape": "sphere", "x": 1.0, "y": 0.0, "z": 0.0},
			},
			"links": []any{map[string]any{"from": "O", "to": "H1"}},
		})
		if !ok || board.Scene3D.AssetKey != "" || len(board.Scene3D.Parts) != 2 || len(board.Scene3D.Links) != 1 {
			t.Fatalf("unexpected procedural result: ok=%v board=%+v", ok, board)
		}
	})

	t.Run("show_3d_model with no asset_key and no usable parts fails", func(t *testing.T) {
		_, ok := BuildBoardContent("show_3d_model", map[string]any{"caption": "empty"})
		if ok {
			t.Fatal("expected ok=false with neither a valid asset_key nor any usable parts")
		}
	})
}

func TestParseScene3DParts(t *testing.T) {
	t.Run("real parts pass through", func(t *testing.T) {
		parts, ok := parseScene3DParts([]any{
			map[string]any{"label": "Carbon", "shape": "sphere", "color": "black", "x": 0.0, "y": 0.0, "z": 0.0},
		})
		if !ok || len(parts) != 1 || parts[0].Shape != "sphere" || parts[0].Size != 1 {
			t.Fatalf("unexpected: ok=%v parts=%+v", ok, parts)
		}
	})

	t.Run("unrecognized shape defaults to sphere instead of dropping the part", func(t *testing.T) {
		parts, ok := parseScene3DParts([]any{
			map[string]any{"label": "Mystery", "shape": "dodecahedron", "x": 0.0, "y": 0.0, "z": 0.0},
		})
		if !ok || len(parts) != 1 || parts[0].Shape != "sphere" {
			t.Fatalf("expected shape fallback to sphere, got: ok=%v parts=%+v", ok, parts)
		}
	})

	t.Run("a part missing its label is dropped", func(t *testing.T) {
		parts, ok := parseScene3DParts([]any{
			map[string]any{"shape": "box", "x": 0.0, "y": 0.0, "z": 0.0},
			map[string]any{"label": "Real", "shape": "box", "x": 0.0, "y": 0.0, "z": 0.0},
		})
		if !ok || len(parts) != 1 || parts[0].Label != "Real" {
			t.Fatalf("expected only the labeled part to survive: ok=%v parts=%+v", ok, parts)
		}
	})

	t.Run("zero usable parts fails", func(t *testing.T) {
		_, ok := parseScene3DParts([]any{map[string]any{"shape": "box"}})
		if ok {
			t.Fatal("expected ok=false when no part has a real label")
		}
	})

	t.Run("wrong top-level type fails", func(t *testing.T) {
		_, ok := parseScene3DParts("not an array")
		if ok {
			t.Fatal("expected ok=false for a non-array argument")
		}
	})

	t.Run("more than maxScene3DParts is capped", func(t *testing.T) {
		items := make([]any, 0, maxScene3DParts+5)
		for i := 0; i < maxScene3DParts+5; i++ {
			items = append(items, map[string]any{"label": "P", "shape": "sphere", "x": 0.0, "y": 0.0, "z": 0.0})
		}
		parts, ok := parseScene3DParts(items)
		if !ok || len(parts) != maxScene3DParts {
			t.Fatalf("expected exactly %d parts, got %d (ok=%v)", maxScene3DParts, len(parts), ok)
		}
	})
}

func TestParseScene3DLinks(t *testing.T) {
	parts := []models.Scene3DPart{
		{Label: "A", Shape: "sphere"},
		{Label: "B", Shape: "sphere"},
	}

	t.Run("a link between two real parts survives", func(t *testing.T) {
		links := parseScene3DLinks([]any{map[string]any{"from": "A", "to": "B"}}, parts)
		if len(links) != 1 {
			t.Fatalf("expected 1 link, got %+v", links)
		}
	})

	t.Run("a dangling reference is dropped, not trusted", func(t *testing.T) {
		links := parseScene3DLinks([]any{map[string]any{"from": "A", "to": "Ghost"}}, parts)
		if len(links) != 0 {
			t.Fatalf("expected the dangling link to be dropped, got %+v", links)
		}
	})

	t.Run("no links is fine, never fails", func(t *testing.T) {
		links := parseScene3DLinks(nil, parts)
		if links != nil {
			t.Fatalf("expected nil/empty links for absent input, got %+v", links)
		}
	})
}

func TestBoardKey(t *testing.T) {
	a, _ := BuildBoardContent("show_working", map[string]any{"title": "Step 1", "lines": []any{"line one", "line two"}})
	b, _ := BuildBoardContent("show_working", map[string]any{"title": "Step 1 (retry)", "lines": []any{"line one", "line two"}})
	c, _ := BuildBoardContent("show_working", map[string]any{"title": "Step 1", "lines": []any{"line one", "a different line"}})

	if BoardKey(a) != BoardKey(b) {
		t.Fatalf("identical lines with a different title should key the same (title varies on repeats in practice): %q != %q", BoardKey(a), BoardKey(b))
	}
	if BoardKey(a) == BoardKey(c) {
		t.Fatalf("genuinely different lines should not key the same: %q", BoardKey(a))
	}

	// The exact real-world shape observed in practice: a restated board
	// with different whitespace on wrapped bullet lines, cut off partway
	// through with a trailing "…" instead of a byte-for-byte repeat.
	original, _ := BuildBoardContent("show_working", map[string]any{
		"title": "Maximizing Profit",
		"lines": []any{
			"Maximizing profit is a fundamental goal of business management. It generally involves two primary strategies:",
			"1.  **Increasing Revenue:** Revenue is the total income generated by sales of goods or services. To increase revenue, businesses might:",
			"    *   Raise prices (if market conditions allow).",
		},
	})
	restatedTruncated, _ := BuildBoardContent("show_working", map[string]any{
		"lines": []any{
			"Maximizing profit is a fundamental goal of business management. It generally involves two primary strategies:",
			"1.  **Increasing Revenue:** Revenue is the total income generated by sales of goods or services. To increase revenue, businesses might:",
			"*   Raise prices (if market conditions allow)…",
		},
	})
	if BoardKey(original) != BoardKey(restatedTruncated) {
		t.Fatalf("a whitespace-varied, truncated restatement should key the same as the original: %q != %q", BoardKey(original), BoardKey(restatedTruncated))
	}
}
