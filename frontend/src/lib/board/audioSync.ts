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
export class BoardAudioSync {
  private queue: ArrayBuffer[] = [];
  private releaseHandler: ((pcm16: ArrayBuffer) => void) | null = null;
  private anchor = 0;
  private releasedMs = 0;
  private done = false;
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
  }

  // Called when the active board unit finishes revealing - let whatever
  // audio is still queued drain out unthrottled rather than making the
  // student wait on an artificial pace once there's nothing left to type.
  markDone() {
    this.done = true;
  }

  // Called on interruption - drops anything not yet released so it can't
  // surface later against an unrelated new anchor.
  clear() {
    this.queue = [];
    this.done = false;
  }

  push(pcm16: ArrayBuffer) {
    this.queue.push(pcm16);
  }

  private tick() {
    if (this.queue.length === 0 || !this.releaseHandler) return;

    if (this.done) {
      while (this.queue.length > 0) {
        this.releaseHandler(this.queue.shift()!);
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
