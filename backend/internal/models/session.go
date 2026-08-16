package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type TutorSession struct {
	ID          bson.ObjectID   `bson:"_id,omitempty" json:"id"`
	OwnerID     bson.ObjectID   `bson:"owner_id" json:"-"`
	Title       string          `bson:"title" json:"title"`
	DocumentIDs []bson.ObjectID `bson:"document_ids,omitempty" json:"document_ids,omitempty"`
	Mode        string          `bson:"mode" json:"mode"`
	Status      string          `bson:"status" json:"status"`
	CreatedAt   time.Time       `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `bson:"updated_at" json:"updated_at"`
	Events      []SessionEvent  `bson:"events" json:"events,omitempty"`

	// GeminiResumptionHandle lets a fresh voice connection for this session
	// resume prior Gemini-side conversation state instead of starting over -
	// backend-internal continuity token, never sent to the frontend.
	GeminiResumptionHandle string `bson:"gemini_resumption_handle,omitempty" json:"-"`

	// LearningPlanID/LearningPlanStepIndex record that this session was
	// started from a specific step of a LearningPlan (see SessionHandler.Create),
	// so the plan detail page can tell "already started teaching this step"
	// apart from "not started yet" and offer Continue vs. Start accordingly.
	// Both are nil for a session created outside the learning-plans flow.
	LearningPlanID        *bson.ObjectID `bson:"learning_plan_id,omitempty" json:"learning_plan_id,omitempty"`
	LearningPlanStepIndex *int           `bson:"learning_plan_step_index,omitempty" json:"learning_plan_step_index,omitempty"`

	// LearningPlanStepTitle/LearningPlanStepObjectives are a snapshot of the
	// step's own content, copied once at session creation (see
	// SessionHandler.Create) rather than looked up again from the plan on
	// every connection - lets live_handler.go ground a voice session's
	// opening turn in exactly this step's topic (see
	// live.LearningPlanStepPrompt) without a second Mongo round-trip to the
	// learning_plans collection. Empty for a session not started from a
	// learning-plan step.
	LearningPlanStepTitle      string   `bson:"learning_plan_step_title,omitempty" json:"-"`
	LearningPlanStepObjectives []string `bson:"learning_plan_step_objectives,omitempty" json:"-"`

	// LearningPlanStepPronunciationNotes is the same kind of snapshot as the
	// two fields above, for the step's phonetic reference (see
	// models.LearningPlanStep.PronunciationNotes) - empty for the vast
	// majority of steps, which aren't about pronouncing a non-English
	// language.
	LearningPlanStepPronunciationNotes string `bson:"learning_plan_step_pronunciation_notes,omitempty" json:"-"`
}

type SessionEvent struct {
	Seq       int           `bson:"seq" json:"seq"`
	Type      string        `bson:"type" json:"type"`
	Role      string        `bson:"role,omitempty" json:"role,omitempty"`
	Text      string        `bson:"text,omitempty" json:"text,omitempty"`
	Board     *BoardContent `bson:"board,omitempty" json:"board,omitempty"`
	Timestamp time.Time     `bson:"timestamp" json:"timestamp"`
}

// BoardContent is one whole "board's worth" of content - a set of typed
// lines (prose/math/code snippets inline), a diagram, a real
// syntax-highlighted code block, or a 3D scene. Kind is the discriminator;
// the frontend renders exactly one of Whiteboard/DiagramBoard/CodeBoard/
// ThreeDBoard based on it. A further kind, "chat", isn't board content at
// all - it's a sentinel PostMessage recognizes and persists as a
// chat_message event instead of a board_update (see session_handlers.go),
// used for a genuine conversational check-in between batches of teaching.
// Message only applies to that kind.
type BoardContent struct {
	Kind    string   `bson:"kind" json:"kind"` // "lines" | "diagram" | "code" | "3d" | "chat" | "clear"
	Title   string   `bson:"title,omitempty" json:"title,omitempty"`
	Lines   []string `bson:"lines,omitempty" json:"lines,omitempty"`
	Mermaid string   `bson:"mermaid,omitempty" json:"mermaid,omitempty"`
	// Code/CodeLanguage back a "code" kind - a real, syntax-highlighted,
	// multi-line code example worth its own board (unlike show_working's
	// existing inline-backtick/fenced-block support within "lines", meant
	// for a short mention rather than a genuine teaching example). No
	// separate explanation field - Theresa narrates around a show_code call
	// with ordinary show_working calls before/after, exactly like
	// draw_diagram doesn't carry its own explanation either.
	Code         string `bson:"code,omitempty" json:"code,omitempty"`
	CodeLanguage string `bson:"code_language,omitempty" json:"code_language,omitempty"`
	// Scene3D backs a "3d" kind - either a real, curated anatomy asset
	// (AssetKey set) or a small procedural scene (Parts/Links) for anything
	// not in the curated set - see Scene3D's own doc comment.
	Scene3D *Scene3D `bson:"scene3d,omitempty" json:"scene3d,omitempty"`
	// Message is decoded from Gemini's streamed JSON response (kind "chat"
	// only) but never persisted as board content - PostMessage reroutes it
	// into a chat_message event's plain Text field instead, so bson is "-".
	Message string `bson:"-" json:"message,omitempty"`
}

// Scene3D is a "3d" board's content - one of two sources, never both:
//   - AssetKey set: a real, curated, properly-licensed anatomy model (see
//     gemini.AnatomyAssets for the closed set of valid keys) - Parts/Links
//     are ignored when this is set.
//   - AssetKey empty: a small, closed, declarative procedural scene -
//     positioned, labeled primitive shapes (Parts), optionally connected
//     (Links) - for geometric shapes, molecules, kid-friendly diagrams, or
//     any anatomy topic not yet in the curated real-asset set. Never
//     free-form Three.js/JS code - that would be a real injection risk,
//     same rationale as every other closed-vocabulary tool in this app.
type Scene3D struct {
	Caption  string        `bson:"caption,omitempty" json:"caption,omitempty"`
	AssetKey string        `bson:"asset_key,omitempty" json:"asset_key,omitempty"`
	Parts    []Scene3DPart `bson:"parts,omitempty" json:"parts,omitempty"`
	Links    []Scene3DLink `bson:"links,omitempty" json:"links,omitempty"`
}

type Scene3DPart struct {
	Label string  `bson:"label" json:"label"`
	Shape string  `bson:"shape" json:"shape"` // "sphere" | "box" | "cylinder" | "cone" | "torus" | "capsule"
	Color string  `bson:"color,omitempty" json:"color,omitempty"`
	Size  float64 `bson:"size,omitempty" json:"size,omitempty"` // relative scale, default 1
	X     float64 `bson:"x" json:"x"`
	Y     float64 `bson:"y" json:"y"`
	Z     float64 `bson:"z" json:"z"`
}

// Scene3DLink connects two Parts by Label (a bond, a simple relationship) -
// From/To must match an existing Part's Label; an unmatched link is dropped
// in Go (see gemini.parseScene3DLinks), never trusted to the frontend
// renderer.
type Scene3DLink struct {
	From  string `bson:"from" json:"from"`
	To    string `bson:"to" json:"to"`
	Label string `bson:"label,omitempty" json:"label,omitempty"`
}
