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
	"time"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"google.golang.org/genai"

	"theresa/backend/internal/config"
	"theresa/backend/internal/gemini"
	"theresa/backend/internal/gemini/live"
	"theresa/backend/internal/models"
)

const maxWSFrameBytes = 512 * 1024

var reconnectBackoff = []time.Duration{500 * time.Millisecond, 1500 * time.Millisecond, 3 * time.Second}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type LiveHandler struct {
	db     *mongo.Database
	cfg    config.Config
	gemini *gemini.Client
}

func NewLiveHandler(db *mongo.Database, cfg config.Config, geminiClient *gemini.Client) *LiveHandler {
	return &LiveHandler{db: db, cfg: cfg, gemini: geminiClient}
}

func (h *LiveHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	session, ok := loadOwnedSession(w, r, h.db)
	if !ok {
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(maxWSFrameBytes)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Passing the session's last-known resumption handle (empty on a
	// genuinely first-ever connection) means ANY new connection for this
	// session - first-ever, a manual page refresh, or a reconnect after the
	// browser's own socket dropped - resumes prior Gemini conversation state
	// if any exists, not just a reconnect within one already-open browser WS.
	liveSession, err := live.Connect(ctx, h.gemini, h.cfg.GeminiLiveModel, session.GeminiResumptionHandle)
	if err != nil {
		conn.WriteJSON(errorMessage("failed to start voice session"))
		log.Printf("live session connect error: %v", err)
		return
	}

	relay := &liveRelay{
		db:               h.db,
		conn:             conn,
		live:             liveSession,
		session:          session,
		seq:              len(session.Events),
		geminiClient:     h.gemini,
		model:            h.cfg.GeminiLiveModel,
		resumptionHandle: session.GeminiResumptionHandle,
	}
	// relay.live may be swapped by attemptReconnect during the connection's
	// lifetime, so cleanup must go through getLive() to close whichever
	// Gemini session is current, not the one captured in this local var.
	defer func() { relay.getLive().Close() }()

	// readPump blocks on conn.ReadMessage() and recvLoop blocks on
	// liveSession.Receive() - neither call is itself context-aware, so
	// whichever side finishes first closes BOTH underlying connections here
	// to unblock the other one's pending read and let its goroutine exit.
	go func() {
		<-ctx.Done()
		conn.Close()
		relay.getLive().Close()
	}()

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

		newSession, err := live.Connect(ctx, rl.geminiClient, rl.model, rl.resumptionHandle)
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
				rl.conn.WriteJSON(audioChunkOutMessage(part.InlineData.Data))
			}
		}
	}

	if content.Interrupted {
		rl.conn.WriteJSON(newWSMessage(wsTypeInterrupted, nil))
	}
	if content.TurnComplete || content.GenerationComplete {
		rl.conn.WriteJSON(newWSMessage(wsTypeTurnComplete, nil))
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
