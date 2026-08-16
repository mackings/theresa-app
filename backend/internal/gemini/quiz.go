package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/genai"
)

// maxQuizGenerationWait bounds the single one-shot structured-output call a
// quiz generation makes - same shape/rationale as maxPlanGenerationWait.
const maxQuizGenerationWait = 45 * time.Second

// maxQuizQuestions caps how many questions one quiz can ever have.
const maxQuizQuestions = 8

const quizSystemInstructionTemplate = `You are writing a short post-lesson multiple-choice quiz
for a tutoring app, based on the conversation below - everything a student was actually just
taught in one tutoring session. Write up to %d multiple-choice questions testing genuine
understanding of what was actually covered in THIS conversation - never generic questions
about the subject in the abstract, never something not actually discussed. If the conversation
covered too little real material for a meaningful quiz, return fewer questions rather than
inventing padding - even one good question is fine, and an empty array is acceptable.

Return a JSON array of question objects, each with:
- "prompt": the question, standalone and clear without needing to re-read the lesson.
- "options": exactly 4 short answer choices, in a sensible (not always correct-answer-first)
  order.
- "correct_index": the 0-based index into "options" of the single correct answer.

Every question must have exactly one unambiguously correct option among its four.`

// GeneratedQuizQuestion mirrors models.QuizQuestion's content fields - kept
// as its own small type here so this package doesn't need to import models
// just to decode a JSON response shape (same pattern as GeneratedStep in
// learningplan.go).
type GeneratedQuizQuestion struct {
	Prompt       string   `json:"prompt"`
	Options      []string `json:"options"`
	CorrectIndex int      `json:"correct_index"`
}

var quizQuestionSchema = &genai.Schema{
	Type:     genai.TypeObject,
	Required: []string{"prompt", "options", "correct_index"},
	Properties: map[string]*genai.Schema{
		"prompt":        {Type: genai.TypeString},
		"options":       {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
		"correct_index": {Type: genai.TypeInteger},
	},
}

// GenerateQuiz is a single one-shot structured-output call, same shape as
// GenerateLearningPlan - a quiz is one coherent structured object generated
// once, not streamed. history is the session's real taught content (see
// HistoryFromEvents), reused as-is rather than a bespoke board-title/line
// extraction: it already caps to maxHistoryEvents, already normalizes every
// board kind into plain text via boardContentAsText, and already includes
// the student's own turns/chat-checkin replies, which carry real signal
// about what was actually emphasized or confusing.
func (c *Client) GenerateQuiz(ctx context.Context, history []*genai.Content) ([]GeneratedQuizQuestion, error) {
	// A definite terminal instruction, since history's last turn could be
	// either role depending on how the lesson ended.
	contents := append(append([]*genai.Content{}, history...), genai.NewContentFromText("Generate the quiz now.", genai.RoleUser))

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(
			fmt.Sprintf(quizSystemInstructionTemplate, maxQuizQuestions), genai.RoleUser),
		ResponseMIMEType: "application/json",
		// Deliberately no array-level MinItems/MaxItems - same confirmed
		// "too many states for serving" API rejection as learningplan.go.
		// dropInvalidQuizQuestions below is the defense-in-depth instead.
		ResponseSchema: &genai.Schema{
			Type:  genai.TypeArray,
			Items: quizQuestionSchema,
		},
	}

	callCtx, cancel := context.WithTimeout(ctx, maxQuizGenerationWait)
	defer cancel()
	resp, err := c.genai.Models.GenerateContent(callCtx, c.textModel, contents, config)
	if err != nil {
		return nil, fmt.Errorf("generate quiz: %w", err)
	}

	text := strings.TrimSpace(resp.Text())
	var questions []GeneratedQuizQuestion
	if err := json.Unmarshal([]byte(text), &questions); err != nil {
		return nil, fmt.Errorf("generate quiz: malformed response: %w", err)
	}
	questions = dropInvalidQuizQuestions(questions)
	if len(questions) > maxQuizQuestions {
		questions = questions[:maxQuizQuestions]
	}
	if len(questions) == 0 {
		return nil, fmt.Errorf("generate quiz: no usable questions produced")
	}
	return questions, nil
}

// dropInvalidQuizQuestions filters out any question that isn't actually
// gradeable - ResponseSchema's Required constraint is a strong hint, not an
// absolute guarantee (same lesson as dropEmptySteps): validates a non-empty
// prompt, exactly 4 options, and an in-range correct_index, since an
// out-of-range index would otherwise silently break grading later.
func dropInvalidQuizQuestions(questions []GeneratedQuizQuestion) []GeneratedQuizQuestion {
	out := make([]GeneratedQuizQuestion, 0, len(questions))
	for _, q := range questions {
		if strings.TrimSpace(q.Prompt) == "" {
			continue
		}
		if len(q.Options) != 4 {
			continue
		}
		if q.CorrectIndex < 0 || q.CorrectIndex >= len(q.Options) {
			continue
		}
		out = append(out, q)
	}
	return out
}
