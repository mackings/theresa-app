package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"theresa/backend/internal/auth"
	"theresa/backend/internal/config"
	"theresa/backend/internal/gemini"
	"theresa/backend/internal/models"
)

// QuizHandler is deliberately scoped to learning-plan sessions only - see
// loadOwnedPlanSession below, which every handler method here goes through.
// A quiz has no standalone ID-based access pattern of its own; it's always
// reached via its owning session, so there's no separate loadOwnedQuiz
// helper - loadOwnedSession (reused as-is) plus a FindOne{"session_id":...}
// is the entire lookup.
type QuizHandler struct {
	db     *mongo.Database
	cfg    config.Config
	gemini *gemini.Client

	// Quiz generation triggers a real Gemini call - throttled per-user like
	// every other Gemini-triggering endpoint in this app.
	createLimiter *auth.RateLimiter
}

func NewQuizHandler(db *mongo.Database, cfg config.Config, geminiClient *gemini.Client) *QuizHandler {
	return &QuizHandler{
		db:            db,
		cfg:           cfg,
		gemini:        geminiClient,
		createLimiter: auth.NewRateLimiter(10, time.Hour),
	}
}

func (h *QuizHandler) quizzes() *mongo.Collection {
	return h.db.Collection("quizzes")
}

// loadOwnedPlanSession loads the session via the existing loadOwnedSession
// (ownership already verified there) and additionally rejects any session
// that didn't originate from a learning-plan step - quizzes are a
// learning-plan-only feature, not something every voice/text session gets
// (explicit product decision, not an oversight). Enforced here, server-side
// - not just by which button the frontend happens to show - so hitting the
// API directly on a plain session can't bypass it either.
func (h *QuizHandler) loadOwnedPlanSession(w http.ResponseWriter, r *http.Request) (models.TutorSession, bool) {
	session, ok := loadOwnedSession(w, r, h.db)
	if !ok {
		return models.TutorSession{}, false
	}
	if session.LearningPlanID == nil {
		writeError(w, http.StatusForbidden, "quizzes are only available for sessions started from a learning plan step")
		return models.TutorSession{}, false
	}
	return session, true
}

// GetOrCreate - POST /api/sessions/{id}/quiz. Idempotent: one quiz per
// session, re-calling this after one already exists just returns it as-is
// rather than regenerating.
func (h *QuizHandler) GetOrCreate(w http.ResponseWriter, r *http.Request) {
	session, ok := h.loadOwnedPlanSession(w, r)
	if !ok {
		return
	}
	if !h.createLimiter.Allow(session.OwnerID.Hex()) {
		writeError(w, http.StatusTooManyRequests, "too many quizzes requested, please try again later")
		return
	}

	var existing models.Quiz
	if err := h.quizzes().FindOne(r.Context(), bson.M{"session_id": session.ID}).Decode(&existing); err == nil {
		writeJSON(w, http.StatusOK, toQuizView(existing))
		return
	}

	if len(session.Events) == 0 {
		writeError(w, http.StatusConflict, "nothing has been taught in this session yet")
		return
	}

	now := time.Now()
	quiz := models.Quiz{
		OwnerID:   session.OwnerID,
		SessionID: session.ID,
		Status:    "generating",
		CreatedAt: now,
		UpdatedAt: now,
	}
	result, err := h.quizzes().InsertOne(r.Context(), quiz)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create quiz")
		return
	}
	quiz.ID = result.InsertedID.(bson.ObjectID)

	go h.generateQuiz(quiz.ID, session.Events)

	writeJSON(w, http.StatusAccepted, toQuizView(quiz))
}

// generateQuiz runs on a detached context (not r.Context(), which dies when
// the handler returns) - same async shape as learningplan_handlers.go's
// generatePlan: retry once on failure, then a final status transition.
func (h *QuizHandler) generateQuiz(quizID bson.ObjectID, events []models.SessionEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	history := gemini.HistoryFromEvents(events)

	generated, err := h.gemini.GenerateQuiz(ctx, history)
	if err != nil {
		log.Printf("quiz %s generation failed, retrying once: %v", quizID.Hex(), err)
		generated, err = h.gemini.GenerateQuiz(ctx, history)
	}
	if err != nil {
		log.Printf("quiz %s generation failed after retry: %v", quizID.Hex(), err)
		h.quizzes().UpdateOne(ctx, bson.M{"_id": quizID}, bson.M{"$set": bson.M{
			"status":        "failed",
			"error_message": "We couldn't generate a quiz for this lesson. Please try again.",
			"updated_at":    time.Now(),
		}})
		return
	}

	questions := make([]models.QuizQuestion, len(generated))
	for i, g := range generated {
		questions[i] = models.QuizQuestion{Prompt: g.Prompt, Options: g.Options, CorrectIndex: g.CorrectIndex}
	}

	h.quizzes().UpdateOne(ctx, bson.M{"_id": quizID}, bson.M{"$set": bson.M{
		"status":     "ready",
		"questions":  questions,
		"updated_at": time.Now(),
	}})
}

// Get - GET /api/sessions/{id}/quiz.
func (h *QuizHandler) Get(w http.ResponseWriter, r *http.Request) {
	session, ok := h.loadOwnedPlanSession(w, r)
	if !ok {
		return
	}

	var quiz models.Quiz
	if err := h.quizzes().FindOne(r.Context(), bson.M{"session_id": session.ID}).Decode(&quiz); err != nil {
		writeError(w, http.StatusNotFound, "no quiz for this session yet")
		return
	}

	writeJSON(w, http.StatusOK, toQuizView(quiz))
}

type submitQuizRequest struct {
	Answers []int `json:"answers"`
}

// Submit - POST /api/sessions/{id}/quiz/submit. No retake-for-better-score
// in this pass - an explicit non-goal, not an oversight; a quiz can only
// ever be attempted once.
func (h *QuizHandler) Submit(w http.ResponseWriter, r *http.Request) {
	session, ok := h.loadOwnedPlanSession(w, r)
	if !ok {
		return
	}

	var quiz models.Quiz
	if err := h.quizzes().FindOne(r.Context(), bson.M{"session_id": session.ID}).Decode(&quiz); err != nil {
		writeError(w, http.StatusNotFound, "no quiz for this session yet")
		return
	}
	if quiz.Status != "ready" {
		writeError(w, http.StatusConflict, "this quiz isn't ready yet")
		return
	}
	if quiz.Attempted {
		writeError(w, http.StatusConflict, "this quiz has already been attempted")
		return
	}

	var req submitQuizRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Answers) != len(quiz.Questions) {
		writeError(w, http.StatusBadRequest, "answers must match the number of questions")
		return
	}

	score := 0
	for i, q := range quiz.Questions {
		// An out-of-range answer just counts as incorrect, never errors -
		// same "trust nothing from the client, degrade gracefully" spirit
		// as the rest of this handler's validation.
		if req.Answers[i] == q.CorrectIndex {
			score++
		}
	}

	quiz.Attempted = true
	quiz.Score = score
	quiz.Answers = req.Answers
	h.quizzes().UpdateOne(r.Context(), bson.M{"_id": quiz.ID}, bson.M{"$set": bson.M{
		"attempted":  true,
		"score":      score,
		"answers":    req.Answers,
		"updated_at": time.Now(),
	}})

	writeJSON(w, http.StatusOK, toQuizView(quiz))
}

type quizQuestionView struct {
	Prompt        string   `json:"prompt"`
	Options       []string `json:"options"`
	CorrectIndex  *int     `json:"correct_index,omitempty"`
	SelectedIndex *int     `json:"selected_index,omitempty"`
	Correct       *bool    `json:"correct,omitempty"`
}

type quizView struct {
	ID         string             `json:"id"`
	SessionID  string             `json:"session_id"`
	Status     string             `json:"status"`
	Attempted  bool               `json:"attempted"`
	Score      int                `json:"score,omitempty"`
	TotalCount int                `json:"total_count,omitempty"`
	Questions  []quizQuestionView `json:"questions,omitempty"`
}

// toQuizView is the single place deciding what's safe to reveal - used
// identically by GetOrCreate/Get/Submit, so the redaction judgment is made
// once, not duplicated three times. When !Attempted, grading fields are
// left nil, so json's omitempty drops them entirely - a pre-submission
// response genuinely cannot leak correct_index, not merely "chooses not to."
func toQuizView(quiz models.Quiz) quizView {
	view := quizView{
		ID:        quiz.ID.Hex(),
		SessionID: quiz.SessionID.Hex(),
		Status:    quiz.Status,
		Attempted: quiz.Attempted,
	}
	if quiz.Status != "ready" {
		return view
	}

	view.TotalCount = len(quiz.Questions)
	view.Questions = make([]quizQuestionView, len(quiz.Questions))
	for i, q := range quiz.Questions {
		qv := quizQuestionView{Prompt: q.Prompt, Options: q.Options}
		if quiz.Attempted {
			correctIndex := q.CorrectIndex
			selectedIndex := -1
			if i < len(quiz.Answers) {
				selectedIndex = quiz.Answers[i]
			}
			correct := selectedIndex == correctIndex
			qv.CorrectIndex = &correctIndex
			qv.SelectedIndex = &selectedIndex
			qv.Correct = &correct
		}
		view.Questions[i] = qv
	}
	if quiz.Attempted {
		view.Score = quiz.Score
	}
	return view
}
