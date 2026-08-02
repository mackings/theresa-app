import { AUDIO_RELEASE_TICK_MS } from "@/lib/board/timing";

const PLAYBACK_SAMPLE_RATE = 24000;
const BYTES_PER_SAMPLE = 2;

function chunkDurationMs(pcm16: ArrayBuffer): number {
  const samples = pcm16.byteLength / BYTES_PER_SAMPLE;
  return (samples / PLAYBACK_SAMPLE_RATE) * 1000;
}

// Paces released audio to wall-clock time since the board started revealing,
// so voice never gets ahead of what's visibly written - it only ever waits
// for the board, absorbing any skew between when audio arrives over the wire
// and when the board unit it narrates actually mounts. Deliberately a plain
// class (not React state/Context): a ~30ms tick driving pure bookkeeping has
// no business causing React re-renders, and this keeps the pacing math
// unit-testable with zero AudioContext dependency.
// How much already-queued audio markDone() lets through unthrottled - see
// markDone's own comment for why this is capped rather than unbounded.
const MARK_DONE_GRACE_MS = 2000;

export class BoardAudioSync {
  private queue: ArrayBuffer[] = [];
  private releaseHandler: ((pcm16: ArrayBuffer) => void) | null = null;
  private anchor = 0;
  private releasedMs = 0;
  private done = false;
  private doneDrainedMs = 0;
  private timer: ReturnType<typeof setInterval> | null = null;

  setReleaseHandler(fn: (pcm16: ArrayBuffer) => void) {
    this.releaseHandler = fn;
  }

  start() {
    if (this.timer) return;
    this.timer = setInterval(() => this.tick(), AUDIO_RELEASE_TICK_MS);
  }

  stop() {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
  }

  // Called when a fresh board unit mounts and starts typing.
  reset() {
    this.anchor = performance.now();
    this.releasedMs = 0;
    this.done = false;
    this.doneDrainedMs = 0;
  }

  // Called when the active board unit finishes revealing - lets a short
  // grace tail of already-queued audio drain out unthrottled (a reasonable
  // bit of trailing audio for the board that just finished) rather than
  // making the student wait on an artificial pace once there's nothing left
  // to type.
  //
  // Deliberately NOT unbounded: a real observed failure was Gemini
  // producing audio (and the next board's own function call) faster than
  // the current board's typewriter could finish revealing it, so a real
  // backlog of audio for the NEXT, not-yet-mounted board was already
  // sitting in this same flat queue by the time this fires. Draining
  // everything here would play that later content immediately, racing far
  // ahead of a board that hasn't started typing yet - exactly the "she
  // says it's on the board but it doesn't show" symptom. Anything beyond
  // this grace window is left queued for the next board's own reset() to
  // pace properly against its own anchor once it actually mounts.
  markDone() {
    this.done = true;
    this.doneDrainedMs = 0;
  }

  // Called on interruption - drops anything not yet released so it can't
  // surface later against an unrelated new anchor.
  clear() {
    this.queue = [];
    this.done = false;
    this.doneDrainedMs = 0;
  }

  push(pcm16: ArrayBuffer) {
    this.queue.push(pcm16);
  }

  private tick() {
    if (this.queue.length === 0 || !this.releaseHandler) return;

    if (this.done) {
      while (this.queue.length > 0 && this.doneDrainedMs < MARK_DONE_GRACE_MS) {
        const next = this.queue.shift()!;
        this.doneDrainedMs += chunkDurationMs(next);
        this.releaseHandler(next);
      }
      return;
    }

    const elapsedMs = performance.now() - this.anchor;
    while (this.queue.length > 0) {
      const next = this.queue[0];
      const duration = chunkDurationMs(next);
      if (this.releasedMs + duration > elapsedMs) break;
      this.queue.shift();
      this.releasedMs += duration;
      this.releaseHandler(next);
    }
  }
}
