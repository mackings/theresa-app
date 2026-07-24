package live

import (
	"context"
	"fmt"

	"google.golang.org/genai"

	"theresa/backend/internal/gemini"
)

// Session wraps a single Gemini Live voice conversation.
type Session struct {
	genai *genai.Session
}

// Connect opens a Gemini Live session. If resumptionHandle is non-empty, it
// asks Gemini to resume that prior conversation's state rather than
// starting fresh (see LiveServerSessionResumptionUpdate); an empty handle
// simply starts a new session, so this is safe to call on the very first
// connect too.
func Connect(ctx context.Context, client *gemini.Client, model, resumptionHandle string) (*Session, error) {
	config := &genai.LiveConnectConfig{
		ResponseModalities: []genai.Modality{genai.ModalityAudio},
		SystemInstruction:  genai.NewContentFromText(PersonaInstruction, genai.RoleUser),
		Tools:              Tools,
		SessionResumption:  &genai.SessionResumptionConfig{Handle: resumptionHandle},
	}

	genaiSession, err := client.Genai().Live.Connect(ctx, model, config)
	if err != nil {
		return nil, fmt.Errorf("connect live session: %w", err)
	}

	return &Session{genai: genaiSession}, nil
}

func (s *Session) SendAudio(pcm16 []byte) error {
	return s.genai.SendRealtimeInput(genai.LiveRealtimeInput{
		Audio: &genai.Blob{Data: pcm16, MIMEType: "audio/pcm;rate=16000"},
	})
}

func (s *Session) SendText(text string) error {
	return s.genai.SendClientContent(genai.LiveClientContentInput{
		Turns: []*genai.Content{genai.NewContentFromText(text, genai.RoleUser)},
	})
}

func (s *Session) Receive() (*genai.LiveServerMessage, error) {
	return s.genai.Receive()
}

// RespondToTool acknowledges a tool call with an honest result - callers
// should pass {"output": "ok"} only when real content actually reached the
// board, or {"error": "..."} otherwise, so the model knows to retry rather
// than believing a no-op call succeeded.
func (s *Session) RespondToTool(callID, name string, response map[string]any) error {
	return s.genai.SendToolResponse(genai.LiveToolResponseInput{
		FunctionResponses: []*genai.FunctionResponse{
			{ID: callID, Name: name, Response: response},
		},
	})
}

func (s *Session) Close() error {
	return s.genai.Close()
}
