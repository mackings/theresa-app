package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"google.golang.org/genai"

	"theresa/backend/internal/auth"
	"theresa/backend/internal/billing"
	"theresa/backend/internal/config"
	"theresa/backend/internal/email"
	"theresa/backend/internal/gemini"
	"theresa/backend/internal/gemini/live"
	"theresa/backend/internal/models"
)

// billingTickInterval is how often accumulated voice usage is converted to
// a credit deduction - frequent enough that the balance reads as "quietly
// ticking down" rather than a large jump at session end, and frequent
// enough that a crash mid-session only loses at most one tick's worth of
// unbilled usage.
const billingTickInterval = 20 * time.Second

// farewellTimeout bounds how long endGracefully waits for Theresa to
// actually speak her out-of-credits goodbye before tearing the connection
// down regardless - a real turn is a couple seconds at most, this just
// guards against Gemini never responding.
const farewellTimeout = 12 * time.Second

const maxWSFrameBytes = 512 * 1024

// connectWithFreshFallback tries to resume with handle, falling back to a
// brand-new connection (empty handle) if that fails. A stored Gemini
// resumption handle can expire server-side on its own schedule, independent
// of anything client-visible - Gemini rejects the whole connection outright
// when that happens ("BidiGenerateContent session expired") rather than
// silently starting fresh, so without this fallback a session with a stale
// handle would refuse to open at all instead of just starting a new
// conversation.
func connectWithFreshFallback(ctx context.Context, client *gemini.Client, model, handle string) (*live.Session, error) {
	session, err := live.Connect(ctx, client, model, handle)
	if err == nil || handle == "" {
		return session, err
	}
	return live.Connect(ctx, client, model, "")
}

var reconnectBackoff = []time.Duration{500 * time.Millisecond, 1500 * time.Millisecond, 3 * time.Second}

type LiveHandler struct {
	db       *mongo.Database
	cfg      config.Config
	gemini   *gemini.Client
	email    *email.Client
	upgrader websocket.Upgrader

	// Each connection opens a real Gemini Live session - throttled per-user
	// to bound cost/abuse from a scripted reconnect loop. Generous enough
	// that normal use (including M5's automatic reconnect-on-drop) never
	// hits it.
	connectLimiter *auth.RateLimiter
}

func NewLiveHandler(db *mongo.Database, cfg config.Config, geminiClient *gemini.Client, emailClient *email.Client) *LiveHandler {
	return &LiveHandler{
		db:             db,
		cfg:            cfg,
		gemini:         geminiClient,
		email:          emailClient,
		connectLimiter: auth.NewRateLimiter(10, time.Minute),
		upgrader: websocket.Upgrader{
			// A WS handshake is a plain GET that carries cookies cross-site
			// under SameSite=None (required for the split frontend/backend
			// domains) - without validating Origin, any page on the internet
			// could open a connection here using a logged-in victim's
			// cookie (cross-site WebSocket hijacking). Only our own
			// frontend origin may open this connection.
			CheckOrigin: func(r *http.Request) bool {
				return r.Header.Get("Origin") == cfg.FrontendURL
			},
		},
	}
}

// firstUnderstoodDocument returns the first attached, fully-processed
// document for grounding a voice session's opening turn - deliberately just
// the first one (teaching from one course at a time keeps the opening
// context simple and bounded) rather than every attached document. Ownership
// isn't re-checked here since session.DocumentIDs only ever contains ids the
// session's own owner attached (see SessionHandler.PostMessage's $addToSet).
func (h *LiveHandler) firstUnderstoodDocument(ctx context.Context, session models.TutorSession) (models.Document, bool) {
	if len(session.DocumentIDs) == 0 {
		return models.Document{}, false
	}
	var doc models.Document
	err := h.db.Collection("documents").FindOne(ctx, bson.M{"_id": session.DocumentIDs[0]}).Decode(&doc)
	if err != nil || doc.Status != "understood" || doc.ExtractedSummary == "" {
		return models.Document{}, false
	}
	return doc, true
}

func (h *LiveHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	if userIDHex, ok := auth.UserIDFromContext(r.Context()); ok && !h.connectLimiter.Allow(userIDHex) {
		writeError(w, http.StatusTooManyRequests, "too many connection attempts, please try again later")
		return
	}

	session, ok := loadOwnedSession(w, r, h.db)
	if !ok {
		return
	}

	var user models.User
	if err := h.db.Collection("users").FindOne(r.Context(), bson.M{"_id": session.OwnerID}).Decode(&user); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(maxWSFrameBytes)

	// Refuse before ever touching Gemini if there's nothing left to spend -
	// free trial exhausted and a zero (or negative, shouldn't happen but
	// guard anyway) credit balance. Saves the cost of opening a Gemini
	// connection just to immediately tear it down.
	if user.FreeTrialSecondsRemaining <= 0 && user.CreditBalanceKobo <= 0 {
		conn.WriteJSON(newWSMessage(wsTypeOutOfCredits, nil))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Passing the session's last-known resumption handle (empty on a
	// genuinely first-ever connection) means ANY new connection for this
	// session - first-ever, a manual page refresh, or a reconnect after the
	// browser's own socket dropped - resumes prior Gemini conversation state
	// if any exists, not just a reconnect within one already-open browser WS.
	liveSession, err := connectWithFreshFallback(ctx, h.gemini, h.cfg.GeminiLiveModel, session.GeminiResumptionHandle)
	if err != nil {
		conn.WriteJSON(errorMessage("failed to start voice session"))
		log.Printf("live session connect error: %v", err)
		return
	}

	relay := &liveRelay{
		db:                        h.db,
		conn:                      conn,
		live:                      liveSession,
		session:                   session,
		seq:                       len(session.Events),
		geminiClient:              h.gemini,
		model:                     h.cfg.GeminiLiveModel,
		resumptionHandle:          session.GeminiResumptionHandle,
		userID:                    user.ID,
		email:                     h.email,
		usdToNGNRate:              h.cfg.USDToNGNRate,
		freeTrialSecondsRemaining: user.FreeTrialSecondsRemaining,
		creditBalanceKobo:         user.CreditBalanceKobo,
		lastBillTime:              time.Now(),
		farewellDone:              make(chan struct{}),
	}
	// relay.live may be swapped by attemptReconnect during the connection's
	// lifetime, so cleanup must go through getLive() to close whichever
	// Gemini session is current, not the one captured in this local var.
	defer func() { relay.getLive().Close() }()

	conn.WriteJSON(creditBalanceMessage(user.CreditBalanceKobo, user.FreeTrialSecondsRemaining))

	// Only the first time this session's content ever reaches the Live/voice
	// side gets an opening message - a non-empty resumption handle means
	// Gemini already has this session's conversation state, so re-sending
	// anything here would duplicate/contradict what it already remembers
	// (covers both a reconnect within one browser session and a page
	// refresh, either of which already has a handle by this point).
	if session.GeminiResumptionHandle == "" {
		isBrandNew := len(session.Events) == 0
		if doc, ok := h.firstUnderstoodDocument(r.Context(), session); ok {
			// A document is attached - ground the opening turn in its
			// summary instead of the generic greeting, so voice mode
			// actually knows what it's teaching instead of starting a
			// random, unrelated conversation. See GreetingWithDocumentPrompt
			// for why this uses the summary rather than attaching the file.
			prompt := live.GreetingWithDocumentPrompt(user.Name, doc.ExtractedSummary, isBrandNew)
			if err := liveSession.SendText(prompt); err != nil {
				log.Printf("failed to send document-grounded opening message: %v", err)
			}
		} else if isBrandNew {
			if err := liveSession.SendText(live.GreetingPrompt(user.Name)); err != nil {
				log.Printf("failed to send opening greeting: %v", err)
			}
		}
	}

	// readPump blocks on conn.ReadMessage() and recvLoop blocks on
	// liveSession.Receive() - neither call is itself context-aware, so
	// whichever side finishes first closes BOTH underlying connections here
	// to unblock the other one's pending read and let its goroutine exit.
	go func() {
		<-ctx.Done()
		conn.Close()
		relay.getLive().Close()
	}()

	go relay.runBillingTicker(ctx, cancel)

	done := make(chan struct{})
	go func() {
		relay.readPump(cancel)
		close(done)
	}()

	relay.recvLoop(ctx, cancel)
	<-done
}

// liveRelay ties one browser WebSocket connection to a Gemini Live session
// and persists events as they arrive. It's the sole writer to this
// session's events for its lifetime, so a local seq counter is safe (unlike
// the concurrent-REST-POST race accepted in the M3 messages endpoint).
//
// live is mutable: attemptReconnect can swap in a fresh Gemini connection
// mid-conversation, so readPump/recvLoop must always go through
// getLive()/setLive() rather than touching the field directly.
type liveRelay struct {
	db      *mongo.Database
	conn    *websocket.Conn
	session models.TutorSession
	seq     int

	geminiClient     *gemini.Client
	model            string
	resumptionHandle string
	titled           bool

	mu   sync.RWMutex
	live *live.Session

	// Billing: userID/email/usdToNGNRate are fixed for the connection's
	// lifetime. The pending counters accumulate real usage between billing
	// ticks and are guarded by billingMu since readPump, recvLoop, and the
	// billing ticker are three separate goroutines all touching them.
	userID       bson.ObjectID
	email        *email.Client
	usdToNGNRate float64

	billingMu                 sync.Mutex
	userAudioBytesPending     int64
	theresaAudioBytesPending  int64
	textInCharsPending        int
	textOutCharsPending       int
	lastBillTime              time.Time
	freeTrialSecondsRemaining int
	creditBalanceKobo         int64

	// Set by endGracefully when the account runs out of credits, so
	// recvLoop knows a subsequent turn_complete is Theresa's farewell
	// finishing (not an ordinary turn) and signals farewellDone.
	awaitingFarewell  atomic.Bool
	farewellDone      chan struct{}
	farewellCloseOnce sync.Once
}

func (rl *liveRelay) getLive() *live.Session {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.live
}

func (rl *liveRelay) setLive(s *live.Session) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.live = s
}

func (rl *liveRelay) readPump(cancel context.CancelFunc) {
	defer cancel()

	for {
		_, data, err := rl.conn.ReadMessage()
		if err != nil {
			return
		}

		var msg wsMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case wsTypeAudioChunkIn:
			var payload audioChunkPayload
			if json.Unmarshal(msg.Payload, &payload) != nil {
				continue
			}
			pcm, err := base64.StdEncoding.DecodeString(payload.AudioB64)
			if err != nil {
				continue
			}
			rl.addUserAudioBytes(len(pcm))
			// A send failure here is non-fatal: a reconnect may be in
			// progress, in which case this chunk is simply dropped (an
			// acceptable loss for a live audio stream) rather than ending
			// the whole connection. Only the browser disconnecting
			// (ReadMessage failing, above) is fatal for this loop.
			if err := rl.getLive().SendAudio(pcm); err != nil {
				log.Printf("send audio failed (dropped): %v", err)
			}

		case wsTypeTextInput:
			var payload textInputPayload
			if json.Unmarshal(msg.Payload, &payload) != nil {
				continue
			}
			if err := rl.getLive().SendText(payload.Text); err != nil {
				log.Printf("send text failed (dropped): %v", err)
			}
		}
	}
}

func (rl *liveRelay) recvLoop(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()

	for {
		msg, err := rl.getLive().Receive()
		if err != nil {
			if rl.attemptReconnect(ctx) {
				continue
			}
			return
		}

		if msg.GoAway != nil {
			if rl.attemptReconnect(ctx) {
				continue
			}
			rl.conn.WriteJSON(errorMessage("voice session ending"))
			return
		}

		if msg.SessionResumptionUpdate != nil && msg.SessionResumptionUpdate.Resumable {
			rl.resumptionHandle = msg.SessionResumptionUpdate.NewHandle
			rl.saveResumptionHandle(rl.resumptionHandle)
		}

		if msg.ServerContent != nil {
			rl.handleServerContent(msg.ServerContent)
		}

		if msg.ToolCall != nil {
			rl.handleToolCall(msg.ToolCall)
		}
	}
}

// attemptReconnect tries to re-establish the Gemini Live connection (resuming
// the prior conversation state if we have a handle for it), retrying with
// backoff. Returns true if a new connection is now in place and recvLoop
// should keep going, false if it should give up.
func (rl *liveRelay) attemptReconnect(ctx context.Context) bool {
	rl.conn.WriteJSON(newWSMessage(wsTypeReconnecting, nil))

	for _, delay := range reconnectBackoff {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(delay):
		}

		newSession, err := connectWithFreshFallback(ctx, rl.geminiClient, rl.model, rl.resumptionHandle)
		if err != nil {
			log.Printf("reconnect attempt failed: %v", err)
			continue
		}

		rl.setLive(newSession)
		rl.conn.WriteJSON(newWSMessage(wsTypeReconnected, nil))
		return true
	}

	return false
}

func (rl *liveRelay) handleServerContent(content *genai.LiveServerContent) {
	if content.ModelTurn != nil {
		for _, part := range content.ModelTurn.Parts {
			if part.InlineData != nil && len(part.InlineData.Data) > 0 {
				rl.addTheresaAudioBytes(len(part.InlineData.Data))
				rl.conn.WriteJSON(audioChunkOutMessage(part.InlineData.Data))
			}
		}
	}

	if content.Interrupted {
		rl.conn.WriteJSON(newWSMessage(wsTypeInterrupted, nil))
	}
	if content.TurnComplete || content.GenerationComplete {
		rl.conn.WriteJSON(newWSMessage(wsTypeTurnComplete, nil))
		if rl.awaitingFarewell.Load() {
			rl.farewellCloseOnce.Do(func() { close(rl.farewellDone) })
		}
	}
}

func (rl *liveRelay) handleToolCall(toolCall *genai.LiveServerToolCall) {
	for _, call := range toolCall.FunctionCalls {
		board, ok := buildBoardContent(call.Name, call.Args)

		response := map[string]any{"output": "ok"}
		if ok {
			rl.conn.WriteJSON(boardUpdateMessage(board))
			rl.appendEvent(models.SessionEvent{Type: "board_update", Role: "assistant", Board: &board})
			rl.maybeSetTitle(board)
			// Rough character counts for the small text portion of this
			// exchange: the board content Gemini generated (billed as text
			// output) and our tiny tool-response acknowledgment (billed as
			// text input). Not exact token counts - see EstimateTextTokens.
			rl.addTextChars(toolResponseChars, boardContentChars(board))
		} else {
			response = map[string]any{"error": fmt.Sprintf(
				"%s call had empty or malformed arguments; retry with real content", call.Name)}
		}

		if err := rl.getLive().RespondToTool(call.ID, call.Name, response); err != nil {
			log.Printf("respond to tool call failed: %v", err)
		}
	}
}

// maybeSetTitle derives and persists a session title from the first board
// actually taught in this voice session, the same way PostMessage does for
// text sessions - otherwise a voice session sits titled "New session"
// forever. Runs at most once per connection (titled latches true whether or
// not a title was actually needed, so a session created with an explicit
// title isn't re-checked on every subsequent board update).

func (rl *liveRelay) maybeSetTitle(board models.BoardContent) {
	if rl.titled {
		return
	}
	rl.titled = true
	if rl.session.Title != "" && rl.session.Title != defaultSessionTitle {
		return
	}

	title := deriveTitle(board, "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rl.db.Collection("sessions").UpdateOne(ctx, bson.M{"_id": rl.session.ID}, bson.M{
		"$set": bson.M{"title": title},
	})
}

// buildBoardContent turns a Gemini function call into a models.BoardContent.
// It reads fields directly off the args map (rather than round-tripping
// through json.Marshal/Unmarshal into a struct, which would silently
// discard type-mismatch errors on a field like "lines") and returns
// ok=false for anything that doesn't yield real, usable content.
func buildBoardContent(name string, args map[string]any) (models.BoardContent, bool) {
	switch name {
	case "show_working":
		lines, ok := coerceStringArray(args["lines"])
		if !ok || len(lines) == 0 {
			return models.BoardContent{}, false
		}
		title, _ := args["title"].(string)
		return models.BoardContent{Kind: "lines", Title: title, Lines: lines}, true

	case "draw_diagram":
		mermaid, _ := args["mermaid"].(string)
		if strings.TrimSpace(mermaid) == "" {
			return models.BoardContent{}, false
		}
		title, _ := args["title"].(string)
		return models.BoardContent{Kind: "diagram", Title: title, Mermaid: mermaid}, true

	default:
		log.Printf("unhandled tool call: %s", name)
		return models.BoardContent{}, false
	}
}

// coerceStringArray handles Gemini's preview-model quirk of occasionally
// sending an array-typed argument JSON-encoded as a string instead of a
// real array.
func coerceStringArray(raw any) ([]string, bool) {
	switch v := raw.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out, len(out) > 0
	case string:
		var arr []string
		if err := json.Unmarshal([]byte(v), &arr); err != nil {
			return nil, false
		}
		return arr, len(arr) > 0
	default:
		return nil, false
	}
}

// saveResumptionHandle persists the latest Gemini resumption handle onto the
// session document - durable past this one connection's lifetime, so a
// later connection (a refresh, or the browser's own socket reconnecting
// after a real network drop) can resume this same conversation rather than
// starting fresh. Fire-and-forget with a short timeout, same shape as
// appendEvent - losing one handle update isn't fatal, the next one supersedes it.
func (rl *liveRelay) saveResumptionHandle(handle string) {
	if handle == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rl.db.Collection("sessions").UpdateOne(ctx, bson.M{"_id": rl.session.ID}, bson.M{
		"$set": bson.M{"gemini_resumption_handle": handle},
	})
}

func (rl *liveRelay) appendEvent(event models.SessionEvent) {
	event.Seq = rl.seq
	event.Timestamp = time.Now()
	rl.seq++

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rl.db.Collection("sessions").UpdateOne(ctx, bson.M{"_id": rl.session.ID}, bson.M{
		"$push": bson.M{"events": event},
		"$set":  bson.M{"updated_at": event.Timestamp},
	})
}

// toolResponseChars is a small fixed estimate for the tiny tool-response
// acknowledgment sent back to Gemini after each show_working/draw_diagram
// call ({"output":"ok"} or a short error string) - not worth measuring
// exactly given how small and constant-shaped it is.
const toolResponseChars = 20

// boardContentChars estimates the character count of what Gemini generated
// for one tool call - the billable "text output" side of a board update.
func boardContentChars(board models.BoardContent) int {
	n := len(board.Title) + len(board.Mermaid)
	for _, line := range board.Lines {
		n += len(line)
	}
	return n
}

func (rl *liveRelay) addUserAudioBytes(n int) {
	rl.billingMu.Lock()
	rl.userAudioBytesPending += int64(n)
	rl.billingMu.Unlock()
}

func (rl *liveRelay) addTheresaAudioBytes(n int) {
	rl.billingMu.Lock()
	rl.theresaAudioBytesPending += int64(n)
	rl.billingMu.Unlock()
}

func (rl *liveRelay) addTextChars(inChars, outChars int) {
	rl.billingMu.Lock()
	rl.textInCharsPending += inChars
	rl.textOutCharsPending += outChars
	rl.billingMu.Unlock()
}

// runBillingTicker periodically converts accumulated audio/text usage into
// a credit deduction. Stops on ctx cancellation (connection ending for any
// other reason) or calls cancel itself if the account runs out of credits
// mid-session, tearing the connection down the same way any other fatal
// condition does.
func (rl *liveRelay) runBillingTicker(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(billingTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if rl.billTick(ctx) {
				rl.endGracefully(ctx, cancel)
				return
			}
		}
	}
}

// endGracefully lets Theresa actually say goodbye - in her own voice - before
// the connection closes, instead of cutting her off silently mid-sentence.
// Tells the frontend immediately (so the mic stops and the UI reflects
// reality right away) while giving Gemini a bounded window to speak the
// farewell turn; recvLoop signals farewellDone as soon as that turn
// completes, and a timeout guards against it never doing so.
func (rl *liveRelay) endGracefully(ctx context.Context, cancel context.CancelFunc) {
	rl.conn.WriteJSON(newWSMessage(wsTypeOutOfCredits, nil))

	rl.awaitingFarewell.Store(true)
	if err := rl.getLive().SendText(live.OutOfCreditsFarewellPrompt); err != nil {
		log.Printf("failed to send out-of-credits farewell prompt: %v", err)
		cancel()
		return
	}

	select {
	case <-rl.farewellDone:
	case <-time.After(farewellTimeout):
	case <-ctx.Done():
	}
	cancel()
}

// billTick converts whatever usage accumulated since the last tick into a
// credit deduction (after first consuming any remaining free trial time),
// and returns true if the account is now out of credits and the session
// should end.
//
// The free trial is spent in plain wall-clock seconds of connection time,
// independent of how much audio actually flowed - simplest to reason about
// ("5 minutes free") and errs generous at the boundary tick rather than
// stingy. Once it's exhausted, real measured audio/text usage from Gemini's
// official per-second/per-token rates takes over.
func (rl *liveRelay) billTick(ctx context.Context) bool {
	now := time.Now()

	rl.billingMu.Lock()
	elapsed := now.Sub(rl.lastBillTime).Seconds()
	rl.lastBillTime = now
	userBytes := rl.userAudioBytesPending
	theresaBytes := rl.theresaAudioBytesPending
	textInChars := rl.textInCharsPending
	textOutChars := rl.textOutCharsPending
	rl.userAudioBytesPending = 0
	rl.theresaAudioBytesPending = 0
	rl.textInCharsPending = 0
	rl.textOutCharsPending = 0
	rl.billingMu.Unlock()

	if elapsed <= 0 {
		return false
	}

	// 16-bit PCM: 2 bytes/sample. Input capture is 16kHz mono, output
	// playback is 24kHz mono (see lib/audio.ts on the frontend).
	userSeconds := float64(userBytes) / (16000 * 2)
	theresaSeconds := float64(theresaBytes) / (24000 * 2)

	billableFraction := 1.0
	if rl.freeTrialSecondsRemaining > 0 {
		if elapsed <= float64(rl.freeTrialSecondsRemaining) {
			rl.freeTrialSecondsRemaining -= int(elapsed)
			rl.saveFreeTrialRemaining()
			rl.conn.WriteJSON(creditBalanceMessage(rl.creditBalanceKobo, rl.freeTrialSecondsRemaining))
			return false
		}
		consumedFromTrial := float64(rl.freeTrialSecondsRemaining)
		billableFraction = (elapsed - consumedFromTrial) / elapsed
		rl.freeTrialSecondsRemaining = 0
		rl.saveFreeTrialRemaining()
	}

	billableUserSeconds := userSeconds * billableFraction
	billableTheresaSeconds := theresaSeconds * billableFraction
	textInTokens := billing.EstimateTextTokens(int(float64(textInChars) * billableFraction))
	textOutTokens := billing.EstimateTextTokens(int(float64(textOutChars) * billableFraction))

	chargeKobo := billing.ChargeKobo(billableUserSeconds, billableTheresaSeconds, textInTokens, textOutTokens, rl.usdToNGNRate)
	if chargeKobo <= 0 {
		return false
	}

	result, err := billing.Deduct(ctx, rl.db, rl.userID, chargeKobo, rl.session.ID, map[string]any{
		"user_audio_seconds":    billableUserSeconds,
		"theresa_audio_seconds": billableTheresaSeconds,
		"text_in_tokens":        textInTokens,
		"text_out_tokens":       textOutTokens,
	})
	if err != nil {
		log.Printf("credit deduction failed: %v", err)
		return false
	}

	rl.creditBalanceKobo = result.RemainingKobo
	rl.conn.WriteJSON(creditBalanceMessage(result.RemainingKobo, rl.freeTrialSecondsRemaining))

	for _, pct := range result.CrossedThresholds {
		rl.conn.WriteJSON(lowCreditsMessage(pct))
		rl.notifyLowCredits(pct, result.RemainingKobo)
	}

	return result.OutOfCredits
}

func (rl *liveRelay) saveFreeTrialRemaining() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rl.db.Collection("users").UpdateOne(ctx, bson.M{"_id": rl.userID}, bson.M{
		"$set": bson.M{"free_trial_seconds_remaining": rl.freeTrialSecondsRemaining},
	})
}

// notifyLowCredits sends the "you've used N% of your credits" email for a
// newly-crossed threshold. Fire-and-forget on a detached context, same
// spirit as the Mongo writes elsewhere in this file - a failed notification
// email isn't worth interrupting a live voice session over.
func (rl *liveRelay) notifyLowCredits(percent int, remainingKobo int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user models.User
	if err := rl.db.Collection("users").FindOne(ctx, bson.M{"_id": rl.userID}).Decode(&user); err != nil {
		return
	}

	remainingNaira := float64(remainingKobo) / 100
	if err := rl.email.SendLowCreditsEmail(ctx, user.Email, user.Name, percent, remainingNaira); err != nil {
		log.Printf("low credits email failed: %v", err)
	}
}
