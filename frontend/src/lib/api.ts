import { SessionEvent } from "@/types/board";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8090";

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

// warmBackend pings the backend's health endpoint without waiting on or
// surfacing the result. Render's free tier spins the backend down after
// inactivity, so the very first real request (e.g. signup) can otherwise
// take 30-60s+ instead of its normal ~1s - calling this the instant a page
// that's about to need the backend mounts gives the cold start a head start
// during the time a real user spends just typing into the form.
export function warmBackend(): void {
  fetch(`${API_URL}/healthz`).catch(() => {});
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const isFormData = init?.body instanceof FormData;

  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      ...(isFormData ? {} : { "Content-Type": "application/json" }),
      // A plain cross-site HTML form can't set this - the backend requires
      // it on state-changing requests specifically so a forged form
      // submission from another site (which would still carry our
      // SameSite=None session cookie) gets rejected before touching any
      // handler logic.
      "X-Requested-With": "XMLHttpRequest",
      ...init?.headers,
    },
  });

  const body = await res.json().catch(() => ({}));

  if (!res.ok) {
    throw new ApiError(res.status, body.error ?? "something went wrong");
  }

  return body as T;
}

// streamSessionMessage posts a message and calls onEvent for each board/user
// event as soon as it's ready, instead of waiting for Gemini's entire
// multi-step answer before anything is visible - the backend streams one
// newline-delimited JSON event per line as it generates each step. A
// mid-stream generation failure arrives as a final {type:"error"} line
// (since the HTTP status is already committed to 200 by the time any real
// content starts streaming) and is surfaced by rejecting with an ApiError,
// same as any other failure this caller might already handle.
export async function streamSessionMessage(
  sessionId: string,
  body: { text?: string; document_id?: string },
  onEvent: (event: SessionEvent) => void
): Promise<void> {
  const res = await fetch(`${API_URL}/api/sessions/${sessionId}/messages`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json", "X-Requested-With": "XMLHttpRequest" },
    body: JSON.stringify(body),
  });

  if (!res.ok || !res.body) {
    const errBody = await res.json().catch(() => ({}));
    throw new ApiError(res.status, errBody.error ?? "something went wrong");
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  function handleLine(raw: string) {
    const line = raw.trim();
    if (!line) return;

    let parsed: Record<string, unknown>;
    try {
      parsed = JSON.parse(line);
    } catch {
      return;
    }

    if (parsed.type === "error") {
      const message = typeof parsed.message === "string" ? parsed.message : "something went wrong";
      throw new ApiError(502, message);
    }
    onEvent(parsed as unknown as SessionEvent);
  }

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });

    let newlineIndex: number;
    while ((newlineIndex = buffer.indexOf("\n")) !== -1) {
      const line = buffer.slice(0, newlineIndex);
      buffer = buffer.slice(newlineIndex + 1);
      handleLine(line);
    }
  }

  handleLine(buffer);
}
