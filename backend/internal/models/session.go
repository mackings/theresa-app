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
}

type SessionEvent struct {
	Seq       int           `bson:"seq" json:"seq"`
	Type      string        `bson:"type" json:"type"`
	Role      string        `bson:"role,omitempty" json:"role,omitempty"`
	Text      string        `bson:"text,omitempty" json:"text,omitempty"`
	Board     *BoardContent `bson:"board,omitempty" json:"board,omitempty"`
	Timestamp time.Time     `bson:"timestamp" json:"timestamp"`
}

// BoardContent is one whole "board's worth" of content - either a set of
// typed lines (prose/math/code) or a single diagram. Kind is the
// discriminator; the frontend renders exactly one of Whiteboard/DiagramBoard
// based on it.
type BoardContent struct {
	Kind    string   `bson:"kind" json:"kind"` // "lines" | "diagram"
	Title   string   `bson:"title,omitempty" json:"title,omitempty"`
	Lines   []string `bson:"lines,omitempty" json:"lines,omitempty"`
	Mermaid string   `bson:"mermaid,omitempty" json:"mermaid,omitempty"`
}
