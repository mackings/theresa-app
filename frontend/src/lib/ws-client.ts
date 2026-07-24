import { BoardContentBlock } from "@/types/board";

export interface LiveSessionHandlers {
  onAudioChunk?: (pcm16: ArrayBuffer) => void;
  onBoardUpdate?: (block: BoardContentBlock) => void;
  onInterrupted?: () => void;
  onTurnComplete?: () => void;
  onError?: (message: string) => void;
  onReconnecting?: () => void;
  onReconnected?: () => void;
  onClose?: () => void;
}

export interface LiveSessionConnection {
  sendAudioChunk: (pcm16: ArrayBuffer) => void;
  sendText: (text: string) => void;
  close: () => void;
}

function wsURL(sessionId: string): string {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8090";
  return apiUrl.replace(/^http/, "ws") + `/ws/session/${sessionId}`;
}

// Backend↔Gemini drops already reconnect transparently (M5) without this
// browser↔backend socket ever needing to reopen. This is the other leg: if
// the browser's OWN connection drops (real network loss, tab suspend,
// laptop sleep), reopen it here automatically, on the same backoff schedule,
// reusing the existing onReconnecting/onReconnected signals (already
// rendering a "Reconnecting…" banner in VoiceControls) - from the student's
// perspective, "the voice connection hiccuped" is one concept either way.
const OUTER_RECONNECT_BACKOFF_MS = [500, 1500, 3000];

export function connectLiveSession(
  sessionId: string,
  handlers: LiveSessionHandlers
): LiveSessionConnection {
  let ws: WebSocket;
  let intentionalClose = false;
  let reconnectAttempt = 0;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  function attach(socket: WebSocket) {
    socket.onmessage = (event: MessageEvent<string>) => {
      // Backend's Payload field is json.RawMessage, which serializes as an
      // embedded JSON value - it's already an object here, not a JSON string.
      const msg: { type: string; payload?: unknown } = JSON.parse(event.data);

      switch (msg.type) {
        case "audio_chunk_out": {
          const { audio_b64 } = msg.payload as { audio_b64: string };
          handlers.onAudioChunk?.(base64ToArrayBuffer(audio_b64));
          break;
        }
        case "board_update":
          handlers.onBoardUpdate?.(msg.payload as BoardContentBlock);
          break;
        case "interrupted":
          handlers.onInterrupted?.();
          break;
        case "turn_complete":
          handlers.onTurnComplete?.();
          break;
        case "error": {
          const { message } = msg.payload as { message: string };
          handlers.onError?.(message);
          break;
        }
        case "reconnecting":
          handlers.onReconnecting?.();
          break;
        case "reconnected":
          handlers.onReconnected?.();
          break;
      }
    };

    socket.onopen = () => {
      if (reconnectAttempt > 0) {
        handlers.onReconnected?.();
      }
      reconnectAttempt = 0;
    };

    socket.onclose = () => {
      if (intentionalClose) {
        handlers.onClose?.();
        return;
      }
      if (reconnectAttempt >= OUTER_RECONNECT_BACKOFF_MS.length) {
        handlers.onClose?.();
        return;
      }
      handlers.onReconnecting?.();
      const delay = OUTER_RECONNECT_BACKOFF_MS[reconnectAttempt];
      reconnectAttempt++;
      reconnectTimer = setTimeout(() => {
        ws = new WebSocket(wsURL(sessionId));
        attach(ws);
      }, delay);
    };
  }

  ws = new WebSocket(wsURL(sessionId));
  attach(ws);

  function whenOpen(send: () => void) {
    if (ws.readyState === WebSocket.OPEN) {
      send();
    }
  }

  return {
    sendAudioChunk: (pcm16) => {
      whenOpen(() => {
        ws.send(
          JSON.stringify({
            type: "audio_chunk_in",
            payload: { audio_b64: arrayBufferToBase64(pcm16) },
          })
        );
      });
    },
    sendText: (text) => {
      whenOpen(() => {
        ws.send(JSON.stringify({ type: "text_input", payload: { text } }));
      });
    },
    close: () => {
      intentionalClose = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      ws.close();
    },
  };
}

function arrayBufferToBase64(buffer: ArrayBuffer): string {
  let binary = "";
  const bytes = new Uint8Array(buffer);
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}
