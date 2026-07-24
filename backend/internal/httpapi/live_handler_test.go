package httpapi

import "testing"

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
			lines, ok := coerceStringArray(tc.input)
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
		board, ok := buildBoardContent("show_working", map[string]any{
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
		board, ok := buildBoardContent("show_working", map[string]any{
			"lines": `["first line", "second line"]`,
		})
		if !ok || len(board.Lines) != 2 {
			t.Fatalf("expected coerced lines, got ok=%v board=%+v", ok, board)
		}
	})

	t.Run("show_working with empty lines fails", func(t *testing.T) {
		_, ok := buildBoardContent("show_working", map[string]any{"lines": []any{}})
		if ok {
			t.Fatal("expected ok=false for empty lines")
		}
	})

	t.Run("draw_diagram with real mermaid", func(t *testing.T) {
		board, ok := buildBoardContent("draw_diagram", map[string]any{
			"mermaid": "graph TD; A-->B;",
		})
		if !ok || board.Kind != "diagram" || board.Mermaid == "" {
			t.Fatalf("unexpected result: ok=%v board=%+v", ok, board)
		}
	})

	t.Run("draw_diagram with blank mermaid fails", func(t *testing.T) {
		_, ok := buildBoardContent("draw_diagram", map[string]any{"mermaid": "   "})
		if ok {
			t.Fatal("expected ok=false for blank mermaid")
		}
	})

	t.Run("unknown tool name fails", func(t *testing.T) {
		_, ok := buildBoardContent("update_board", map[string]any{"step": 1})
		if ok {
			t.Fatal("expected ok=false for unhandled tool name")
		}
	})
}
