package httpapi

import (
	"context"
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

	// PostMessage triggers a real, paid Gemini call every time - this is a
	// per-user throttle so a scripted loop (or a compromised session) can't
	// run up unbounded API cost. Generous enough that no real conversational
	// use should ever hit it.
	messageLimiter *auth.RateLimiter
}

func NewSessionHandler(db *mongo.Database, cfg config.Config, geminiClient *gemini.Client) *SessionHandler {
	return &SessionHandler{
		db:             db,
		cfg:            cfg,
		gemini:         geminiClient,
		messageLimiter: auth.NewRateLimiter(20, time.Minute),
	}
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

// wsTicketTTL is short deliberately - the ticket only needs to survive the
// brief gap between this call and the WebSocket upgrade request right after
// it (plus however many of the reconnect loop's own retries happen before
// one succeeds), not a real session lifetime.
const wsTicketTTL = 60 * time.Second

// IssueWSTicket mints a short-lived, single-purpose token for the voice
// WebSocket's own auth (see auth.RequireAuthCookieOrTicket). That connection
// goes directly to the backend's own origin - cross-site from the frontend's,
// unlike every other call which now goes through the frontend's proxy - so it
// can't rely on the session cookie the way proxied REST calls can. Ownership
// is enforced the same way as every other session endpoint (loadOwnedSession)
// before a ticket is ever minted.
func (h *SessionHandler) IssueWSTicket(w http.ResponseWriter, r *http.Request) {
	session, ok := loadOwnedSession(w, r, h.db)
	if !ok {
		return
	}

	var user struct {
		TokenVersion int `bson:"token_version"`
	}
	opts := options.FindOne().SetProjection(bson.M{"token_version": 1})
	if err := h.db.Collection("users").FindOne(r.Context(), bson.M{"_id": session.OwnerID}, opts).Decode(&user); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue ticket")
		return
	}

	ticket, err := auth.MintToken(session.OwnerID.Hex(), user.TokenVersion, h.cfg.JWTSecret, wsTicketTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue ticket")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"ticket": ticket})
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

	if !h.messageLimiter.Allow(session.OwnerID.Hex()) {
		writeError(w, http.StatusTooManyRequests, "too many requests, please slow down")
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

		h.sessions().UpdateOne(r.Context(), bson.M{"_id": session.ID}, bson.M{
			"$addToSet": bson.M{"document_ids": id},
		})
	}

	// Everything above can still fail with a normal HTTP status - nothing's
	// been written to the response yet. From here on we're committed to a
	// 200 streaming response: each event is sent to the browser as its own
	// newline-delimited JSON line the moment it's ready (matching how voice
	// mode already streams board updates over its WebSocket), instead of
	// buffering the entire multi-step answer before the caller sees anything.
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	encoder := json.NewEncoder(w)
	writeLine := func(v any) error {
		if err := encoder.Encode(v); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}

	seq := len(session.Events)
	needsTitle := session.Title == "" || session.Title == defaultSessionTitle

	if req.Text != "" {
		event := models.SessionEvent{Seq: seq, Type: "user_text", Role: "user", Text: req.Text, Timestamp: time.Now()}
		seq++
		persistEvent(h.db, session.ID, event)
		if writeLine(event) != nil {
			return
		}
	}

	err := h.gemini.GenerateBoardStream(r.Context(), boardReq, func(block models.BoardContent) error {
		event := models.SessionEvent{Seq: seq, Type: "board_update", Role: "assistant", Board: &block, Timestamp: time.Now()}
		seq++

		var title string
		if needsTitle {
			title = deriveTitle(block, req.Text)
			needsTitle = false
		}
		persistEvent(h.db, session.ID, event, title)

		return writeLine(event)
	})

	if err != nil {
		writeLine(map[string]string{"type": "error", "message": "failed to generate teaching content"})
	}
}

// persistEvent appends a single event to a session's history, optionally
// setting a derived title in the same update. Fire-and-forget on a detached
// timeout context (mirrors the WS voice relay's appendEvent) rather than the
// request context, since a client disconnecting mid-stream shouldn't cancel
// the write for content Gemini has already produced.
func persistEvent(db *mongo.Database, sessionID bson.ObjectID, event models.SessionEvent, title ...string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	setFields := bson.M{"updated_at": event.Timestamp}
	if len(title) > 0 && title[0] != "" {
		setFields["title"] = title[0]
	}

	db.Collection("sessions").UpdateOne(ctx, bson.M{"_id": sessionID}, bson.M{
		"$push": bson.M{"events": event},
		"$set":  setFields,
	})
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
