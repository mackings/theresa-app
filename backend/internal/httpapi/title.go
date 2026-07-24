package httpapi

import (
	"strings"

	"theresa/backend/internal/models"
)

const defaultSessionTitle = "New session"

const maxTitleLen = 60

// deriveTitle produces a short, human-meaningful session title from the
// first board actually taught in a session, so a student can recognize what
// a session was about from the sidebar/dashboard instead of every session
// reading "New session" forever. Prefers the board's own title (Gemini
// sometimes supplies a short heading), falls back to its first line, then to
// the user's own message text.
func deriveTitle(board models.BoardContent, fallbackText string) string {
	if t := strings.TrimSpace(board.Title); t != "" && looksLikeTitle(t) {
		return truncateTitle(t)
	}
	if len(board.Lines) > 0 {
		if t := strings.TrimSpace(board.Lines[0]); t != "" && looksLikeTitle(t) {
			return truncateTitle(t)
		}
	}
	if t := strings.TrimSpace(fallbackText); t != "" {
		return truncateTitle(t)
	}
	return defaultSessionTitle
}

// looksLikeTitle rejects text that's clearly not fit to be a title - raw
// JSON leaking through (starts with "{"/"[") from a malformed model
// response, defense in depth alongside GenerateBoard's own sanitization of
// that same failure mode.
func looksLikeTitle(s string) bool {
	return !strings.HasPrefix(s, "{") && !strings.HasPrefix(s, "[")
}

func truncateTitle(s string) string {
	if len(s) <= maxTitleLen {
		return s
	}
	// Truncate at a word boundary so titles don't cut off mid-word.
	cut := s[:maxTitleLen]
	if idx := strings.LastIndex(cut, " "); idx > 20 {
		cut = cut[:idx]
	}
	return strings.TrimSpace(cut) + "…"
}
