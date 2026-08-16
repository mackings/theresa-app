package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// LearningPlan is a Gemini-generated, paced sequence of teaching steps -
// either grounded in an uploaded document or a plain stated goal (exactly
// one of DocumentID/Goal is set, validated in the handler, not here).
// Steps are embedded in one document rather than a separate collection,
// matching TutorSession's existing events[]-embedded-in-one-doc convention:
// a plan realistically has single-digit-to-dozens of steps, nowhere near
// the scale that would justify splitting it out.
type LearningPlan struct {
	ID            bson.ObjectID      `bson:"_id,omitempty" json:"id"`
	OwnerID       bson.ObjectID      `bson:"owner_id" json:"-"`
	Title         string             `bson:"title" json:"title"`
	Goal          string             `bson:"goal,omitempty" json:"goal,omitempty"`
	DocumentID    *bson.ObjectID     `bson:"document_id,omitempty" json:"document_id,omitempty"`
	DurationValue int                `bson:"duration_value" json:"duration_value"`
	DurationUnit  string             `bson:"duration_unit" json:"duration_unit"` // "days" | "weeks" | "months"
	Status        string             `bson:"status" json:"status"`               // "generating" | "ready" | "failed"
	ErrorMessage  string             `bson:"error_message,omitempty" json:"error_message,omitempty"`
	Steps         []LearningPlanStep `bson:"steps,omitempty" json:"steps,omitempty"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

// LearningPlanStep is one paced unit of the plan. SessionID is set the
// first time the student actually starts teaching for this step (see
// SessionHandler.Create's learning-plan linkage), so the plan detail page
// can offer "Continue" instead of "Start" once a session already exists.
type LearningPlanStep struct {
	Index      int            `bson:"index" json:"index"`
	Label      string         `bson:"label" json:"label"` // "Day 1", "Week 2"
	Title      string         `bson:"title" json:"title"`
	Objectives []string       `bson:"objectives,omitempty" json:"objectives,omitempty"`
	SessionID  *bson.ObjectID `bson:"session_id,omitempty" json:"session_id,omitempty"`

	// PronunciationNotes is a phonetic reference (IPA or plain respelling)
	// for this step's specific sounds/words, only ever set when the step is
	// actually about pronouncing a non-English language - see
	// gemini.GenerateLearningPlan's prompt. Read by the voice persona as an
	// authoritative pronunciation guide (see live.LearningPlanStepPrompt)
	// instead of letting the live audio model guess at unfamiliar sounds.
	PronunciationNotes string `bson:"pronunciation_notes,omitempty" json:"pronunciation_notes,omitempty"`
}
