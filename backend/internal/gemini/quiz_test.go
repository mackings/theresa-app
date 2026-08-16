package gemini

import "testing"

func TestDropInvalidQuizQuestions(t *testing.T) {
	valid := GeneratedQuizQuestion{
		Prompt:       "What does show_working do?",
		Options:      []string{"A", "B", "C", "D"},
		CorrectIndex: 1,
	}

	cases := []struct {
		name    string
		input   []GeneratedQuizQuestion
		wantLen int
	}{
		{"a fully valid question survives", []GeneratedQuizQuestion{valid}, 1},
		{
			"blank prompt is dropped",
			[]GeneratedQuizQuestion{{Prompt: "   ", Options: valid.Options, CorrectIndex: 0}},
			0,
		},
		{
			"wrong option count is dropped",
			[]GeneratedQuizQuestion{{Prompt: "Q", Options: []string{"A", "B", "C"}, CorrectIndex: 0}},
			0,
		},
		{
			"negative correct_index is dropped",
			[]GeneratedQuizQuestion{{Prompt: "Q", Options: valid.Options, CorrectIndex: -1}},
			0,
		},
		{
			"out-of-range correct_index is dropped",
			[]GeneratedQuizQuestion{{Prompt: "Q", Options: valid.Options, CorrectIndex: 4}},
			0,
		},
		{
			"one bad question doesn't drop a good one alongside it",
			[]GeneratedQuizQuestion{valid, {Prompt: "", Options: nil, CorrectIndex: 0}},
			1,
		},
		{"empty input stays empty", []GeneratedQuizQuestion{}, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := dropInvalidQuizQuestions(tc.input)
			if len(out) != tc.wantLen {
				t.Fatalf("len = %d, want %d (out=%+v)", len(out), tc.wantLen, out)
			}
		})
	}
}
