package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/genai"
)

// maxPlanGenerationWait bounds the single one-shot structured-output call a
// plan generation makes - generous compared to maxSingleCallWait since this
// produces a whole multi-step plan in one response, not one board.
const maxPlanGenerationWait = 45 * time.Second

// maxPlanSteps caps how many steps a plan can ever have, regardless of how
// large a duration the user picks - a careless "12 months" request would
// otherwise ask Gemini to plan an unusably long list. One step per rough
// unit of the requested duration is the natural size; this is a hard
// ceiling, not a target.
const maxPlanSteps = 60

const learningPlanSystemInstructionTemplate = `You are designing a paced learning plan for a
tutoring app. Given either an uploaded document or a plain stated goal, and a requested
duration (a number of days, weeks, or months), break the material into that many
sequential steps - roughly one step per day/week/month requested, never more than %d
steps total regardless of how large the duration is.

Return a JSON array of step objects, oldest-first, each with:
- "label": a short pacing label like "Day 1" or "Week 2" - matching the requested duration
  unit, numbered sequentially starting at 1.
- "title": a few words naming the step's topic - never a full sentence, never the
  explanation itself.
- "objectives": 2-4 short bullet-point strings, each a concrete, specific thing the student
  will learn or be able to do by the end of that step - not vague filler like "understand
  the basics."
- "pronunciation_notes" (only when relevant, otherwise omit entirely): if - and only if -
  this step involves teaching the sounds, letters, or words of a language other than
  English (a French/Spanish/Yoruba/Mandarin/etc. lesson, not just a lesson conducted in
  English), write a short, precise phonetic reference for the specific letters/sounds/words
  this step covers, using IPA or a clear plain-language respelling (e.g. "French 'B' is
  pronounced /be/, like 'beh' - NOT like the English word 'bee'"). This is read by a voice
  tutor as an authoritative pronunciation reference for this exact lesson, so be concrete
  and specific to the actual content of this step, not a generic overview of the language.
  Leave this field out entirely for any step that isn't specifically about pronouncing a
  non-English language's sounds/words - most steps won't have it.

Keep steps genuinely sequential and building on each other, not a random reshuffling of
the same topic. If grounded in a document, follow the document's actual structure and
content, not a generic template for the subject. If given only a stated goal with no
document, use your own knowledge of how that subject is normally taught, in a sensible
beginner-to-further-progress order.`

type LearningPlanRequest struct {
	Goal          string
	FileURI       string
	FileMimeType  string
	DurationValue int
	DurationUnit  string
}

// GeneratedStep mirrors models.LearningPlanStep's content fields (not Index/
// SessionID, which are the handler's responsibility to fill in) - kept as
// its own small type here so this package doesn't need to import models
// just to decode a JSON response shape.
type GeneratedStep struct {
	Label              string   `json:"label"`
	Title              string   `json:"title"`
	Objectives         []string `json:"objectives"`
	PronunciationNotes string   `json:"pronunciation_notes"`
}

var learningPlanStepSchema = &genai.Schema{
	Type:     genai.TypeObject,
	Required: []string{"label", "title"},
	Properties: map[string]*genai.Schema{
		"label":               {Type: genai.TypeString},
		"title":               {Type: genai.TypeString},
		"objectives":          {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
		"pronunciation_notes": {Type: genai.TypeString},
	},
}

// GenerateLearningPlan is a single one-shot structured-output call - the
// opposite shape from GenerateBoardStream's incremental function-calling:
// a plan is one coherent structured object generated once, not streamed
// board-by-board, so ResponseSchema JSON mode (which board.go deliberately
// moved away from for streaming/reliability reasons that don't apply here)
// is the right fit.
func (c *Client) GenerateLearningPlan(ctx context.Context, req LearningPlanRequest) ([]GeneratedStep, error) {
	promptText := fmt.Sprintf("Requested duration: %d %s.", req.DurationValue, req.DurationUnit)
	if req.Goal != "" {
		promptText += fmt.Sprintf(" Goal: %s", req.Goal)
	}

	parts := []*genai.Part{{Text: promptText}}
	if req.FileURI != "" {
		parts = append(parts, &genai.Part{FileData: &genai.FileData{FileURI: req.FileURI, MIMEType: req.FileMimeType}})
	}
	contents := []*genai.Content{genai.NewContentFromParts(parts, genai.RoleUser)}

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(
			fmt.Sprintf(learningPlanSystemInstructionTemplate, maxPlanSteps), genai.RoleUser),
		ResponseMIMEType: "application/json",
		// Deliberately no MinItems/MaxItems on this array schema - Gemini's
		// API rejects a schema combining an array length bound with a nested
		// object schema as "too many states for serving" (confirmed live:
		// Error 400 INVALID_ARGUMENT). The step-count ceiling is enforced by
		// the prompt text (maxPlanSteps, above) and, defensively, by
		// truncating the parsed response in code below - the same
		// "prompt compliance alone isn't reliable, verify/repair in code"
		// approach board.go already uses for other Gemini quirks.
		ResponseSchema: &genai.Schema{
			Type:  genai.TypeArray,
			Items: learningPlanStepSchema,
		},
	}

	callCtx, cancel := context.WithTimeout(ctx, maxPlanGenerationWait)
	defer cancel()
	resp, err := c.genai.Models.GenerateContent(callCtx, c.textModel, contents, config)
	if err != nil {
		return nil, fmt.Errorf("generate learning plan: %w", err)
	}

	text := strings.TrimSpace(resp.Text())
	var steps []GeneratedStep
	if err := json.Unmarshal([]byte(text), &steps); err != nil {
		return nil, fmt.Errorf("generate learning plan: malformed response: %w", err)
	}
	steps = dropEmptySteps(steps)
	if len(steps) > maxPlanSteps {
		steps = steps[:maxPlanSteps]
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("generate learning plan: no usable steps produced")
	}
	return steps, nil
}

// dropEmptySteps filters out any step missing its required label/title -
// ResponseSchema's Required constraint is a strong hint, not an absolute
// guarantee against a degenerate response, so this is the same
// defense-in-depth spirit as board.go's Gemini-quirk repair functions
// rather than trusting schema compliance blindly.
func dropEmptySteps(steps []GeneratedStep) []GeneratedStep {
	out := make([]GeneratedStep, 0, len(steps))
	for _, s := range steps {
		if strings.TrimSpace(s.Label) == "" || strings.TrimSpace(s.Title) == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}
