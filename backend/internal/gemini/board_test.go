package gemini

import (
	"reflect"
	"testing"
)

func TestRepairOrphanedListMarkers(t *testing.T) {
	cases := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "no lists, unchanged",
			input: []string{"The equation is $x^2 - 5x + 6 = 0$", "This is a quadratic."},
			want:  []string{"The equation is $x^2 - 5x + 6 = 0$", "This is a quadratic."},
		},
		{
			// The exact real-world garbled shape observed in practice:
			// "1" split off from the FIRST item's line entirely.
			name: "bare numeric markers merged into following line",
			input: []string{
				"We look for two numbers that:",
				"1",
				"Multiply to give c",
				"2",
				"Add up to give b",
			},
			want: []string{
				"We look for two numbers that:",
				"1. Multiply to give c",
				"2. Add up to give b",
			},
		},
		{
			name:  "bare dash marker merged",
			input: []string{"Options:", "-", "First option", "-", "Second option"},
			want:  []string{"Options:", "- First option", "- Second option"},
		},
		{
			name:  "already well-formed list, unchanged",
			input: []string{"We look for two numbers that:", "1. Multiply to give c", "2. Add up to give b"},
			want:  []string{"We look for two numbers that:", "1. Multiply to give c", "2. Add up to give b"},
		},
		{
			name:  "trailing digit on a real value is NOT touched (ambiguous, deliberately left alone)",
			input: []string{"$a = 1$ 2", "$b = -5$ 3"},
			want:  []string{"$a = 1$ 2", "$b = -5$ 3"},
		},
		{
			name:  "bare marker as the very last line has nothing to merge into, left as-is",
			input: []string{"Some text", "1"},
			want:  []string{"Some text", "1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RepairOrphanedListMarkers(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}
