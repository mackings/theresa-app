package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Document struct {
	ID               bson.ObjectID `bson:"_id,omitempty" json:"id"`
	OwnerID          bson.ObjectID `bson:"owner_id" json:"-"`
	Filename         string        `bson:"filename" json:"filename"`
	MimeType         string        `bson:"mime_type" json:"mime_type"`
	GridFSFileID     bson.ObjectID `bson:"gridfs_file_id" json:"-"`
	SizeBytes        int64         `bson:"size_bytes" json:"size_bytes"`
	Status           string        `bson:"status" json:"status"`
	ExtractedSummary string        `bson:"extracted_summary,omitempty" json:"extracted_summary,omitempty"`
	GeminiFileURI    string        `bson:"gemini_file_uri,omitempty" json:"-"`
	CreatedAt        time.Time     `bson:"created_at" json:"created_at"`
	ProcessedAt      *time.Time    `bson:"processed_at,omitempty" json:"processed_at,omitempty"`
	ErrorMessage     string        `bson:"error_message,omitempty" json:"error_message,omitempty"`
}
