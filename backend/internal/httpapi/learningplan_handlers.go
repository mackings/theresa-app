package httpapi

import (
	"context"
	"encoding/json"
	"log"
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

var allowedDurationUnits = map[string]bool{"days": true, "weeks": true, "months": true}

// LearningPlanHandler is deliberately self-contained - its own collection,
// its own ownership-check helper (loadOwnedPlan below, a small intentional
// duplicate of loadOwnedSession's shape rather than a reuse), its own
// generation goroutine - so this feature doesn't tangle with the existing
// session/document handlers beyond the one documented linkage point in
// SessionHandler.Create.
type LearningPlanHandler struct {
	db     *mongo.Database
	cfg    config.Config
	gemini *gemini.Client

	// Plan generation triggers a real Gemini call - throttled per-user like
	// every other Gemini-triggering endpoint in this app.
	createLimiter *auth.RateLimiter
}

func NewLearningPlanHandler(db *mongo.Database, cfg config.Config, geminiClient *gemini.Client) *LearningPlanHandler {
	return &LearningPlanHandler{
		db:            db,
		cfg:           cfg,
		gemini:        geminiClient,
		createLimiter: auth.NewRateLimiter(10, time.Hour),
	}
}

func (h *LearningPlanHandler) plans() *mongo.Collection {
	return h.db.Collection("learning_plans")
}

func (h *LearningPlanHandler) documents() *mongo.Collection {
	return h.db.Collection("documents")
}

type createLearningPlanRequest struct {
	Title         string `json:"title"`
	Goal          string `json:"goal"`
	DocumentID    string `json:"document_id"`
	DurationValue int    `json:"duration_value"`
	DurationUnit  string `json:"duration_unit"`
}

func (h *LearningPlanHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !h.createLimiter.Allow(ownerID.Hex()) {
		writeError(w, http.StatusTooManyRequests, "too many plans created, please try again later")
		return
	}

	var req createLearningPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if (req.Goal == "") == (req.DocumentID == "") {
		writeError(w, http.StatusBadRequest, "provide exactly one of goal or document_id")
		return
	}
	if !allowedDurationUnits[req.DurationUnit] {
		writeError(w, http.StatusBadRequest, "duration_unit must be \"days\", \"weeks\", or \"months\"")
		return
	}
	if req.DurationValue < 1 || req.DurationValue > 52 {
		writeError(w, http.StatusBadRequest, "duration_value must be between 1 and 52")
		return
	}

	var docID *bson.ObjectID
	var doc models.Document
	if req.DocumentID != "" {
		id, err := bson.ObjectIDFromHex(req.DocumentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid document_id")
			return
		}
		if err := h.documents().FindOne(r.Context(), bson.M{"_id": id, "owner_id": ownerID}).Decode(&doc); err != nil {
			writeError(w, http.StatusNotFound, "document not found")
			return
		}
		if doc.Status != "understood" {
			writeError(w, http.StatusConflict, "document is still being processed")
			return
		}
		docID = &id
	}

	title := req.Title
	if title == "" {
		title = "New learning plan"
	}

	now := time.Now()
	plan := models.LearningPlan{
		OwnerID:       ownerID,
		Title:         title,
		Goal:          req.Goal,
		DocumentID:    docID,
		DurationValue: req.DurationValue,
		DurationUnit:  req.DurationUnit,
		Status:        "generating",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	result, err := h.plans().InsertOne(r.Context(), plan)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create learning plan")
		return
	}
	plan.ID = result.InsertedID.(bson.ObjectID)

	go h.generatePlan(plan.ID, gemini.LearningPlanRequest{
		Goal:          req.Goal,
		FileURI:       doc.GeminiFileURI,
		FileMimeType:  doc.MimeType,
		DurationValue: req.DurationValue,
		DurationUnit:  req.DurationUnit,
	})

	writeJSON(w, http.StatusAccepted, plan)
}

// generatePlan runs on a detached context (not r.Context(), which dies when
// the handler returns) - same async shape as document_handlers.go's
// processDocument: retry once on failure, then a final status transition to
// "ready" (with real steps) or "failed" (with a generic message; the real
// error is only logged server-side).
func (h *LearningPlanHandler) generatePlan(planID bson.ObjectID, req gemini.LearningPlanRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	generated, err := h.gemini.GenerateLearningPlan(ctx, req)
	if err != nil {
		log.Printf("learning plan %s generation failed, retrying once: %v", planID.Hex(), err)
		generated, err = h.gemini.GenerateLearningPlan(ctx, req)
	}
	if err != nil {
		log.Printf("learning plan %s generation failed after retry: %v", planID.Hex(), err)
		h.plans().UpdateOne(ctx, bson.M{"_id": planID}, bson.M{"$set": bson.M{
			"status":        "failed",
			"error_message": "We couldn't generate a plan for that. Please try again or adjust your goal/duration.",
			"updated_at":    time.Now(),
		}})
		return
	}

	steps := make([]models.LearningPlanStep, len(generated))
	for i, g := range generated {
		steps[i] = models.LearningPlanStep{
			Index:              i,
			Label:              g.Label,
			Title:              g.Title,
			Objectives:         g.Objectives,
			PronunciationNotes: g.PronunciationNotes,
		}
	}

	h.plans().UpdateOne(ctx, bson.M{"_id": planID}, bson.M{"$set": bson.M{
		"status":     "ready",
		"steps":      steps,
		"updated_at": time.Now(),
	}})
}

func (h *LearningPlanHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}})
	cursor, err := h.plans().Find(r.Context(), bson.M{"owner_id": ownerID}, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list learning plans")
		return
	}
	defer cursor.Close(r.Context())

	plans := []models.LearningPlan{}
	if err := cursor.All(r.Context(), &plans); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list learning plans")
		return
	}

	writeJSON(w, http.StatusOK, plans)
}

func (h *LearningPlanHandler) Get(w http.ResponseWriter, r *http.Request) {
	plan, ok := loadOwnedPlan(w, r, h.db)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (h *LearningPlanHandler) Delete(w http.ResponseWriter, r *http.Request) {
	plan, ok := loadOwnedPlan(w, r, h.db)
	if !ok {
		return
	}
	// No cascade to sessions already linked to this plan's steps - same
	// "no cascade" precedent as SessionHandler.Delete not cascading to
	// documents. Those sessions simply keep their learning_plan_id pointing
	// at a plan that no longer exists, harmless since nothing dereferences
	// it from that direction.
	h.plans().DeleteOne(r.Context(), bson.M{"_id": plan.ID})
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// loadOwnedPlan is a deliberate, small duplicate of session_handlers.go's
// loadOwnedSession - kept separate rather than shared/exported so the
// learning-plans resource stays genuinely self-contained. 404 (not 403) on
// any mismatch or not-found, same information-hiding rationale as its
// session counterpart.
func loadOwnedPlan(w http.ResponseWriter, r *http.Request, db *mongo.Database) (models.LearningPlan, bool) {
	ownerID, ok := ownerIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return models.LearningPlan{}, false
	}

	planID, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "learning plan not found")
		return models.LearningPlan{}, false
	}

	var plan models.LearningPlan
	if err := db.Collection("learning_plans").FindOne(r.Context(), bson.M{"_id": planID, "owner_id": ownerID}).Decode(&plan); err != nil {
		writeError(w, http.StatusNotFound, "learning plan not found")
		return models.LearningPlan{}, false
	}

	return plan, true
}
