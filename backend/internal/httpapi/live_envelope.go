package httpapi

import (
	"encoding/base64"
	"encoding/json"

	"theresa/backend/internal/models"
)

// wsMessage is the browser<->backend WebSocket envelope. It's deliberately
// simpler than Gemini's own wire format - the frontend never sees genai
// types directly.
type wsMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

const (
	wsTypeAudioChunkIn  = "audio_chunk_in"  // client -> server: {audio_b64} 16kHz PCM16
	wsTypeAudioChunkOut = "audio_chunk_out" // server -> client: {audio_b64} 24kHz PCM16
	wsTypeBoardUpdate   = "board_update"    // server -> client: BoardContent
	wsTypeInterrupted   = "interrupted"     // server -> client
	wsTypeTurnComplete  = "turn_complete"   // server -> client
	wsTypeTextInput     = "text_input"      // client -> server: {text}
	wsTypeError         = "error"           // server -> client: {message}
	wsTypeReconnecting  = "reconnecting"    // server -> client
	wsTypeReconnected   = "reconnected"     // server -> client
)

type audioChunkPayload struct {
	AudioB64 string `json:"audio_b64"`
}

type textInputPayload struct {
	Text string `json:"text"`
}

type errorPayload struct {
	Message string `json:"message"`
}

func newWSMessage(msgType string, payload any) wsMessage {
	raw, _ := json.Marshal(payload)
	return wsMessage{Type: msgType, Payload: raw}
}

func audioChunkOutMessage(pcm []byte) wsMessage {
	return newWSMessage(wsTypeAudioChunkOut, audioChunkPayload{AudioB64: base64.StdEncoding.EncodeToString(pcm)})
}

func boardUpdateMessage(block models.BoardContent) wsMessage {
	return newWSMessage(wsTypeBoardUpdate, block)
}

func errorMessage(message string) wsMessage {
	return newWSMessage(wsTypeError, errorPayload{Message: message})
}
