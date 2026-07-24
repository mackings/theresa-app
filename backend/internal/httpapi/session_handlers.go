package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"theresa/backend/internal/auth"
	"theresa/backend/internal/config"
	"theresa/backend/internal/gemini"
	"theresa/backend/internal/models"
)

type SessionHandler struct {
	db     *mongo.Database
	cfg    config.Config
	gemini *gemini.Client
}

func NewSessionHandler(db *mongo.Database, cfg config.Config, geminiClient *gemini.Client) *SessionHandler {
	return &SessionHandler{db: db, cfg: cfg, gemini: geminiClient}
}

func (h *SessionHandler) sessions() *mongo.Collection {
	return h.db.Collection("sessions")
}

func (h *SessionHandler) documents() *mongo.Collection {
	return h.db.Collection("documents")
}

func ownerIDFromContext(r *http.Request) (bson.ObjectID, bool) {
	userIDHex, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		return bson.ObjectID{}, false
	}
	ownerID, err := bson.ObjectIDFromHex(userIDHex)
	return ownerID, err == nil
}

type createSessionRequest struct {
	Title       string   `json:"title"`
	DocumentIDs []string `json:"document_ids"`
	Mode        string   `json:"mode"`
}

func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createSessionRequest
	json.NewDecoder(r.Body).Decode(&req)

	title := req.Title
	if title == "" {
		title = defaultSessionTitle
	}
	mode := req.Mode
	if mode == "" {
		mode = "text"
	}
	if mode != "text" && mode != "voice" {
		writeError(w, http.StatusBadRequest, "mode must be 'text' or 'voice'")
		return
	}

	var docIDs []bson.ObjectID
	for _, idHex := range req.DocumentIDs {
		id, err := bson.ObjectIDFromHex(idHex)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid document_id")
			return
		}
		docIDs = append(docIDs, id)
	}

	now := time.Now()
	session := models.TutorSession{
		OwnerID:     ownerID,
		Title:       title,
		DocumentIDs: docIDs,
		Mode:        mode,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
		Events:      []models.SessionEvent{},
	}

	result, err := h.sessions().InsertOne(r.Context(), session)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	session.ID = result.InsertedID.(bson.ObjectID)

	writeJSON(w, http.StatusCreated, session)
}

func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}}).SetProjection(bson.M{"events": 0})
	cursor, err := h.sessions().Find(r.Context(), bson.M{"owner_id": ownerID}, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}
	defer cursor.Close(r.Context())

	sessions := []models.TutorSession{}
	if err := cursor.All(r.Context(), &sessions); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	writeJSON(w, http.StatusOK, sessions)
}

func (h *SessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	session, ok := loadOwnedSession(w, r, h.db)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, session)
}

type updateModeRequest struct {
	Mode string `json:"mode"`
}

// UpdateMode lets a session be switched between text and voice after
// creation, so a student who started by uploading a document (text mode by
// default) can move to voice without losing the conversation - the session's
// events/title/document are unchanged, only how the browser renders the
// right-hand panel and which live connection it opens.
func (h *SessionHandler) UpdateMode(w http.ResponseWriter, r *http.Request) {
	session, ok := loadOwnedSession(w, r, h.db)
	if !ok {
		return
	}

	var req updateModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Mode != "text" && req.Mode != "voice" {
		writeError(w, http.StatusBadRequest, "mode must be 'text' or 'voice'")
		return
	}

	now := time.Now()
	if _, err := h.sessions().UpdateOne(r.Context(), bson.M{"_id": session.ID}, bson.M{
		"$set": bson.M{"mode": req.Mode, "updated_at": now},
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update session")
		return
	}

	session.Mode = req.Mode
	session.UpdatedAt = now
	writeJSON(w, http.StatusOK, session)
}

func (h *SessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	session, ok := loadOwnedSession(w, r, h.db)
	if !ok {
		return
	}

	if _, err := h.sessions().DeleteOne(r.Context(), bson.M{"_id": session.ID}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete session")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

type postMessageRequest struct {
	Text       string `json:"text"`
	DocumentID string `json:"document_id"`
}

func (h *SessionHandler) PostMessage(w http.ResponseWriter, r *http.Request) {
	session, ok := loadOwnedSession(w, r, h.db)
	if !ok {
		return
	}

	var req postMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Text == "" && req.DocumentID == "" {
		writeError(w, http.StatusBadRequest, "text or document_id is required")
		return
	}

	boardReq := gemini.BoardRequest{Text: req.Text}
	var docObjID bson.ObjectID
	hasDoc := false

	if req.DocumentID != "" {
		id, err := bson.ObjectIDFromHex(req.DocumentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid document_id")
			return
		}

		var doc models.Document
		if err := h.documents().FindOne(r.Context(), bson.M{"_id": id, "owner_id": session.OwnerID}).Decode(&doc); err != nil {
			writeError(w, http.StatusNotFound, "document not found")
			return
		}
		if doc.Status != "understood" {
			writeError(w, http.StatusConflict, "document is still processing, try again shortly")
			return
		}

		boardReq.FileURI = doc.GeminiFileURI
		boardReq.FileMimeType = doc.MimeType
		docObjID = id
		hasDoc = true
	}

	blocks, err := h.gemini.GenerateBoard(r.Context(), boardReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to generate teaching content")
		return
	}

	seq := len(session.Events)
	now := time.Now()
	var newEvents []models.SessionEvent

	if req.Text != "" {
		newEvents = append(newEvents, models.SessionEvent{
			Seq: seq, Type: "user_text", Role: "user", Text: req.Text, Timestamp: now,
		})
		seq++
	}
	for _, block := range blocks {
		block := block
		newEvents = append(newEvents, models.SessionEvent{
			Seq: seq, Type: "board_update", Role: "assistant", Board: &block, Timestamp: now,
		})
		seq++
	}

	setFields := bson.M{"updated_at": now}
	if (session.Title == "" || session.Title == defaultSessionTitle) && len(blocks) > 0 {
		setFields["title"] = deriveTitle(blocks[0], req.Text)
	}

	update := bson.M{
		"$push": bson.M{"events": bson.M{"$each": newEvents}},
		"$set":  setFields,
	}
	if hasDoc {
		update["$addToSet"] = bson.M{"document_ids": docObjID}
	}

	if _, err := h.sessions().UpdateOne(r.Context(), bson.M{"_id": session.ID}, update); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save conversation")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"events": newEvents})
}

// loadOwnedSession loads the session named by the "id" URL param, scoped to
// the authenticated caller. Shared by SessionHandler (REST) and LiveHandler
// (WebSocket) so both surfaces enforce the same ownership check.
func loadOwnedSession(w http.ResponseWriter, r *http.Request, db *mongo.Database) (models.TutorSession, bool) {
	ownerID, ok := ownerIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return models.TutorSession{}, false
	}

	sessionID, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return models.TutorSession{}, false
	}

	var session models.TutorSession
	if err := db.Collection("sessions").FindOne(r.Context(), bson.M{"_id": sessionID, "owner_id": ownerID}).Decode(&session); err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return models.TutorSession{}, false
	}

	return session, true
}
