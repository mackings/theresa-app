package storage

import (
	"context"
	"io"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Store struct {
	bucket *mongo.GridFSBucket
}

func NewStore(db *mongo.Database) *Store {
	return &Store{bucket: db.GridFSBucket()}
}

func (s *Store) Upload(ctx context.Context, filename, contentType string, r io.Reader) (bson.ObjectID, error) {
	opts := options.GridFSUpload().SetMetadata(bson.D{{Key: "contentType", Value: contentType}})
	return s.bucket.UploadFromStream(ctx, filename, r, opts)
}

func (s *Store) Download(ctx context.Context, fileID bson.ObjectID) (io.ReadCloser, error) {
	return s.bucket.OpenDownloadStream(ctx, fileID)
}
