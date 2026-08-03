package httpapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"theresa/backend/internal/auth"
	"theresa/backend/internal/config"
	"theresa/backend/internal/gemini"
	"theresa/backend/internal/models"
	"theresa/backend/internal/storage"
)

var allowedDocumentMimeTypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
}

type DocumentHandler struct {
	db      *mongo.Database
	cfg     config.Config
	storage *storage.Store
	gemini  *gemini.Client

	// Each upload triggers real Gemini file-understanding work - throttled
	// per-user to bound cost from a scripted upload loop.
	uploadLimiter *auth.RateLimiter
}

func NewDocumentHandler(db *mongo.Database, cfg config.Config, geminiClient *gemini.Client) *DocumentHandler {
	return &DocumentHandler{
		db:            db,
		cfg:           cfg,
		storage:       storage.NewStore(db),
		gemini:        geminiClient,
		uploadLimiter: auth.NewRateLimiter(10, time.Hour),
	}
}

func (h *DocumentHandler) documents() *mongo.Collection {
	return h.db.Collection("documents")
}

func (h *DocumentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userIDHex, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ownerID, err := bson.ObjectIDFromHex(userIDHex)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if !h.uploadLimiter.Allow(userIDHex) {
		writeError(w, http.StatusTooManyRequests, "too many uploads, please try again later")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxUploadSizeBytes)
	if err := r.ParseMultipartForm(h.cfg.MaxUploadSizeBytes); err != nil {
		// http.MaxBytesReader reports a *http.MaxBytesError specifically when
		// the body was too big - distinguished from any other malformed-
		// multipart-data error so a student who hits the real size limit is
		// told exactly that (with the actual limit, so they know what to do
		// about it) instead of a generic, ambiguous "invalid upload" that
		// reads the same whether their file was too big or just corrupted.
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			limitMB := h.cfg.MaxUploadSizeBytes / (1024 * 1024)
			writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf(
				"That file is larger than the %dMB upload limit. Please use a smaller file, or split it into parts, and try again.",
				limitMB,
			))
			return
		}
		writeError(w, http.StatusBadRequest, "We couldn't read that upload - please try again.")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file")
		return
	}
	defer file.Close()

	buf, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read file")
		return
	}

	mimeType := http.DetectContentType(buf)
	if !allowedDocumentMimeTypes[mimeType] {
		writeError(w, http.StatusBadRequest, "unsupported file type: only PDF and JPEG/PNG images are allowed")
		return
	}

	gridFSFileID, err := h.storage.Upload(r.Context(), header.Filename, mimeType, bytes.NewReader(buf))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store file")
		return
	}

	doc := models.Document{
		OwnerID:      ownerID,
		Filename:     header.Filename,
		MimeType:     mimeType,
		GridFSFileID: gridFSFileID,
		SizeBytes:    int64(len(buf)),
		Status:       "uploaded",
		CreatedAt:    time.Now(),
	}

	result, err := h.documents().InsertOne(r.Context(), doc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create document")
		return
	}
	doc.ID = result.InsertedID.(bson.ObjectID)

	h.documents().UpdateOne(r.Context(), bson.M{"_id": doc.ID}, bson.M{"$set": bson.M{"status": "processing"}})
	doc.Status = "processing"

	go h.processDocument(doc.ID, buf, mimeType, header.Filename)

	writeJSON(w, http.StatusAccepted, doc)
}

func (h *DocumentHandler) processDocument(docID bson.ObjectID, buf []byte, mimeType, filename string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	now := time.Now()
	fileURI, summary, err := h.gemini.UnderstandDocument(ctx, bytes.NewReader(buf), mimeType, filename)
	if err != nil {
		// A transient hiccup (a dropped connection mid-upload to Gemini's
		// Files API, a slow readiness poll, an occasional API blip) is far
		// more common than a genuinely bad file, and was going uncaught here
		// even though every other Gemini call in this app already retries
		// once for exactly this reason (see GenerateBoardStream's retry in
		// session_handlers.go) - this was the one call site that didn't,
		// which is the likely cause of small, valid documents occasionally
		// failing outright. bytes.NewReader is recreated since the first
		// attempt already consumed the original reader.
		log.Printf("document %s processing failed, retrying once: %v", docID.Hex(), err)
		fileURI, summary, err = h.gemini.UnderstandDocument(ctx, bytes.NewReader(buf), mimeType, filename)
	}
	if err != nil {
		// The real error (which can include raw Gemini SDK/API internals)
		// is logged server-side only - the document's own owner sees a
		// generic, actionable message instead, not implementation detail.
		log.Printf("document %s processing failed after retry: %v", docID.Hex(), err)
		h.documents().UpdateOne(ctx, bson.M{"_id": docID}, bson.M{"$set": bson.M{
			"status":        "failed",
			"error_message": "We couldn't process this document. Please try again or use a different file.",
			"processed_at":  now,
		}})
		return
	}

	h.documents().UpdateOne(ctx, bson.M{"_id": docID}, bson.M{"$set": bson.M{
		"status":            "understood",
		"gemini_file_uri":   fileURI,
		"extracted_summary": summary,
		"processed_at":      now,
	}})
}

func (h *DocumentHandler) List(w http.ResponseWriter, r *http.Request) {
	userIDHex, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ownerID, err := bson.ObjectIDFromHex(userIDHex)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cursor, err := h.documents().Find(r.Context(), bson.M{"owner_id": ownerID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}
	defer cursor.Close(r.Context())

	docs := []models.Document{}
	if err := cursor.All(r.Context(), &docs); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}

	writeJSON(w, http.StatusOK, docs)
}

func (h *DocumentHandler) Get(w http.ResponseWriter, r *http.Request) {
	doc, ok := h.loadOwnedDocument(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *DocumentHandler) Download(w http.ResponseWriter, r *http.Request) {
	doc, ok := h.loadOwnedDocument(w, r)
	if !ok {
		return
	}

	stream, err := h.storage.Download(r.Context(), doc.GridFSFileID)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", doc.MimeType)
	io.Copy(w, stream)
}

func (h *DocumentHandler) loadOwnedDocument(w http.ResponseWriter, r *http.Request) (models.Document, bool) {
	userIDHex, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return models.Document{}, false
	}
	ownerID, err := bson.ObjectIDFromHex(userIDHex)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return models.Document{}, false
	}

	docID, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "document not found")
		return models.Document{}, false
	}

	var doc models.Document
	if err := h.documents().FindOne(r.Context(), bson.M{"_id": docID, "owner_id": ownerID}).Decode(&doc); err != nil {
		writeError(w, http.StatusNotFound, "document not found")
		return models.Document{}, false
	}

	return doc, true
}
