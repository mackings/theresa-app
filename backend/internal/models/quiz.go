package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Quiz is one auto-generated, auto-graded multiple-choice quiz for exactly
// one TutorSession - optional, never a gate on progress, and only ever
// created for a session that was started from a learning-plan step (see
// httpapi.QuizHandler, which rejects any other session before a Quiz row
// is ever inserted).
//
// Questions/Answers/Score/CorrectIndex are all json:"-" - stricter than
// this codebase's usual per-field redaction (see User.PasswordHash,
// Document.GeminiFileURI): even an accidental raw writeJSON(w, quiz) leaks
// nothing. Every real API response is built through an explicit view
// projection in quiz_handlers.go instead, so what's safe to reveal is
// decided in exactly one place.
type Quiz struct {
	ID           bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	OwnerID      bson.ObjectID  `bson:"owner_id" json:"-"`
	SessionID    bson.ObjectID  `bson:"session_id" json:"session_id"`
	Status       string         `bson:"status" json:"status"` // "generating" | "ready" | "failed"
	ErrorMessage string         `bson:"error_message,omitempty" json:"-"`
	Questions    []QuizQuestion `bson:"questions,omitempty" json:"-"`
	Attempted    bool           `bson:"attempted" json:"-"`
	Score        int            `bson:"score,omitempty" json:"-"`
	// Answers is the student's picks, index-aligned with Questions - set
	// once, on Submit.
	Answers   []int     `bson:"answers,omitempty" json:"-"`
	CreatedAt time.Time `bson:"created_at" json:"-"`
	UpdatedAt time.Time `bson:"updated_at" json:"-"`
}

type QuizQuestion struct {
	Prompt  string   `bson:"prompt" json:"-"`
	Options []string `bson:"options" json:"-"`
	// CorrectIndex never leaves the backend before the quiz is Attempted -
	// see quiz_handlers.go's toQuizView.
	CorrectIndex int `bson:"correct_index" json:"-"`
}
