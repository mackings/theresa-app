"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { AlertCircle, Mic, MessageCircle, MicOff, PhoneOff, Send, WifiOff } from "lucide-react";
import { connectLiveSession, LiveSessionConnection } from "@/lib/ws-client";
import { startMicCapture, MicCapture, PlaybackQueue } from "@/lib/audio";
import { BoardAudioSync } from "@/lib/board/audioSync";
import { SpeakingOrb, OrbState } from "@/components/voice/SpeakingOrb";
import { IconButton } from "@/components/ui/Button";
import { Pill } from "@/components/ui/Pill";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { BoardContentBlock } from "@/types/board";
import { setActiveVoiceSession } from "@/lib/activeVoiceSession";

const LOW_CREDITS_TOAST_MS = 6000;

export function VoiceControls({
  sessionId,
  onBoardUpdate,
  audioSync,
}: {
  sessionId: string;
  onBoardUpdate: (block: BoardContentBlock) => void;
  audioSync: BoardAudioSync;
}) {
  const router = useRouter();
  const [micOn, setMicOn] = useState(false);
  const [speaking, setSpeaking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [textInput, setTextInput] = useState("");
  const [reconnecting, setReconnecting] = useState(false);
  const [showTextInput, setShowTextInput] = useState(false);
  const [outOfCredits, setOutOfCredits] = useState(false);
  const [lowCreditsToast, setLowCreditsToast] = useState<number | null>(null);
  const [confirmingEnd, setConfirmingEnd] = useState(false);

  const connectionRef = useRef<LiveSessionConnection | null>(null);
  const micRef = useRef<MicCapture | null>(null);
  const playbackRef = useRef<PlaybackQueue | null>(null);

  useEffect(() => {
    const playback = new PlaybackQueue();
    playbackRef.current = playback;

    // Audio arrives off the wire as fast as the network allows, which can
    // outrun the board's typewriter reveal - audioSync gates release of each
    // chunk to wall-clock time since the board actually started typing, so
    // playback here only ever receives audio it's cleared to play now.
    audioSync.setReleaseHandler((pcm16) => {
      setSpeaking(true);
      playback.enqueue(pcm16);
    });
    audioSync.start();
    setActiveVoiceSession(true);

    const connection = connectLiveSession(sessionId, {
      onAudioChunk: (pcm16) => {
        audioSync.push(pcm16);
      },
      onBoardUpdate,
      onInterrupted: () => {
        playback.stopAll();
        audioSync.clear();
        setSpeaking(false);
      },
      onTurnComplete: () => setSpeaking(false),
      onError: (message) => setError(message),
      onReconnecting: () => setReconnecting(true),
      onReconnected: () => setReconnecting(false),
      onOutOfCredits: () => {
        setOutOfCredits(true);
        setActiveVoiceSession(false);
        micRef.current?.stop();
        micRef.current = null;
        setMicOn(false);
      },
      onLowCredits: (percentUsed) => setLowCreditsToast(percentUsed),
    });
    connectionRef.current = connection;

    return () => {
      setActiveVoiceSession(false);
      connection.close();
      playback.close();
      audioSync.stop();
      micRef.current?.stop();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId]);

  useEffect(() => {
    if (lowCreditsToast === null) return;
    const timer = setTimeout(() => setLowCreditsToast(null), LOW_CREDITS_TOAST_MS);
    return () => clearTimeout(timer);
  }, [lowCreditsToast]);

  async function toggleMic() {
    if (outOfCredits) return;
    if (micOn) {
      micRef.current?.stop();
      micRef.current = null;
      setMicOn(false);
      return;
    }

    try {
      const mic = await startMicCapture((chunk) => {
        connectionRef.current?.sendAudioChunk(chunk);
      });
      micRef.current = mic;
      setMicOn(true);
    } catch {
      setError("microphone permission is required for voice mode");
    }
  }

  function sendText(e: React.FormEvent) {
    e.preventDefault();
    if (!textInput.trim()) return;
    connectionRef.current?.sendText(textInput);
    setTextInput("");
  }

  const status = speaking ? "Speaking…" : micOn ? "Listening…" : "Hey, tap the mic to start talking";
  const orbState: OrbState = reconnecting
    ? "reconnecting"
    : speaking
      ? "speaking"
      : micOn
        ? "listening"
        : "idle";

  return (
    <div className="relative flex h-full flex-col">
      {lowCreditsToast !== null && (
        <Link
          href="/credits"
          className={`absolute left-1/2 top-4 z-10 flex -translate-x-1/2 items-center gap-2 whitespace-nowrap rounded-[var(--radius-full)] px-4 py-2 text-sm font-medium shadow-[var(--shadow-md)] transition-opacity hover:opacity-90 ${
            lowCreditsToast >= 95
              ? "bg-[var(--color-danger)] text-white"
              : "border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] text-[var(--color-text-primary)]"
          }`}
        >
          <AlertCircle className="h-4 w-4 shrink-0" />
          {lowCreditsToast >= 95
            ? "Almost out of credits - add more to keep going"
            : `You've used ${lowCreditsToast}% of your voice credits`}
        </Link>
      )}

      <div className="flex flex-1 flex-col items-center justify-center gap-3 p-4 sm:gap-5 sm:p-6">
        <div className="sm:hidden">
          <SpeakingOrb state={orbState} size={44} />
        </div>
        <div className="hidden sm:block">
          <SpeakingOrb state={orbState} />
        </div>
        <Pill>Theresa</Pill>
        <p className="max-w-xs text-center text-base font-semibold text-[var(--color-text-primary)] sm:text-xl">
          {outOfCredits ? "Out of voice credits" : status}
        </p>
        {outOfCredits && (
          <Link
            href="/credits"
            className="rounded-[var(--radius-full)] bg-[var(--color-accent)] px-4 py-2 text-sm font-semibold text-[var(--color-accent-foreground)] shadow-[var(--shadow-xs)] transition-opacity hover:opacity-90"
          >
            Add credits to keep going
          </Link>
        )}
      </div>

      <div className="p-4">
        {reconnecting && (
          <p className="mb-2 flex items-center gap-1.5 rounded-[var(--radius-md)] bg-[var(--color-surface-hover)] px-3 py-1.5 text-xs text-[var(--color-text-secondary)]">
            <WifiOff className="h-3.5 w-3.5 shrink-0" />
            Reconnecting…
          </p>
        )}
        {error && (
          <p className="mb-2 flex items-center gap-1.5 rounded-[var(--radius-md)] bg-[var(--color-surface-hover)] px-3 py-1.5 text-xs text-[var(--color-danger)]">
            <AlertCircle className="h-3.5 w-3.5 shrink-0" />
            {error}
          </p>
        )}

        {showTextInput && (
          <form
            onSubmit={sendText}
            className="mb-3 flex items-center gap-2 rounded-[var(--radius-full)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] py-1 pl-4 pr-1.5 shadow-[var(--shadow-xs)]"
          >
            <input
              type="text"
              value={textInput}
              onChange={(e) => setTextInput(e.target.value)}
              placeholder="Or type instead..."
              autoFocus
              className="flex-1 bg-transparent py-1.5 text-sm text-[var(--color-text-primary)] outline-none"
            />
            <IconButton
              type="submit"
              variant="primary"
              aria-label="Send message"
              className="h-8 w-8 rounded-[var(--radius-full)]"
            >
              <Send className="h-4 w-4" />
            </IconButton>
          </form>
        )}

        <div className="flex items-center justify-center gap-6">
          <IconButton
            variant="secondary"
            aria-label={showTextInput ? "Hide text input" : "Type instead"}
            onClick={() => setShowTextInput((v) => !v)}
            className="h-12 w-12 rounded-[var(--radius-full)]"
          >
            <MessageCircle className="h-5 w-5" />
          </IconButton>
          <button
            type="button"
            onClick={toggleMic}
            aria-label={micOn ? "Mute microphone" : "Unmute microphone"}
            className="flex h-16 w-16 items-center justify-center rounded-[var(--radius-full)] bg-[var(--color-text-primary)] text-[var(--color-bg)] shadow-[var(--shadow-md)] transition-transform hover:scale-105"
          >
            {/* Icon reflects mic state, not a "what tapping does" hint - a
                slash-through mic means "you are muted right now", matching
                how every real calling app (phone, Zoom, Meet) reads this
                button, not the inverse "tap to mute" affordance. */}
            {micOn ? <Mic className="h-6 w-6" /> : <MicOff className="h-6 w-6" />}
          </button>
          <button
            type="button"
            onClick={() => setConfirmingEnd(true)}
            aria-label="End session"
            className="flex h-12 w-12 items-center justify-center rounded-[var(--radius-full)] bg-[var(--color-danger)] text-white shadow-[var(--shadow-sm)] transition-transform hover:scale-105"
          >
            <PhoneOff className="h-5 w-5" />
          </button>
        </div>
      </div>

      <ConfirmDialog
        open={confirmingEnd}
        title="End this voice session?"
        description="Theresa will stop talking and you'll be taken back to the dashboard."
        confirmLabel="End session"
        cancelLabel="Stay"
        onConfirm={() => router.push("/dashboard")}
        onCancel={() => setConfirmingEnd(false)}
      />
    </div>
  );
}
