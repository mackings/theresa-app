export interface MicCapture {
  stop: () => void;
}

// Requests a 16kHz AudioContext so the browser resamples mic input at the
// OS/hardware level where supported. Not all browsers/devices honor a
// requested sample rate - if a target device doesn't, this would need a
// manual resampling fallback in the worklet (unverified cross-browser,
// flagged in the M4 plan; check Safari specifically).
export async function startMicCapture(
  onChunk: (pcm16: ArrayBuffer) => void
): Promise<MicCapture> {
  const audioContext = new AudioContext({ sampleRate: 16000 });
  await audioContext.audioWorklet.addModule("/audio-worklet-processor.js");

  const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  const source = audioContext.createMediaStreamSource(stream);
  const workletNode = new AudioWorkletNode(audioContext, "pcm-capture-processor");

  workletNode.port.onmessage = (event: MessageEvent<ArrayBuffer>) => {
    onChunk(event.data);
  };

  source.connect(workletNode);

  return {
    stop: () => {
      workletNode.port.onmessage = null;
      source.disconnect();
      workletNode.disconnect();
      stream.getTracks().forEach((track) => track.stop());
      audioContext.close();
    },
  };
}

function pcm16ToFloat32(pcm16: ArrayBuffer): Float32Array<ArrayBuffer> {
  const int16 = new Int16Array(pcm16);
  const float32 = new Float32Array(new ArrayBuffer(int16.length * 4));
  for (let i = 0; i < int16.length; i++) {
    float32[i] = int16[i] / (int16[i] < 0 ? 0x8000 : 0x7fff);
  }
  return float32;
}

const PLAYBACK_SAMPLE_RATE = 24000;

export class PlaybackQueue {
  private audioContext: AudioContext;
  private nextStartTime = 0;

  constructor() {
    this.audioContext = new AudioContext({ sampleRate: PLAYBACK_SAMPLE_RATE });
  }

  enqueue(pcm16: ArrayBuffer) {
    const float32 = pcm16ToFloat32(pcm16);
    const buffer = this.audioContext.createBuffer(1, float32.length, PLAYBACK_SAMPLE_RATE);
    buffer.copyToChannel(float32, 0);

    const source = this.audioContext.createBufferSource();
    source.buffer = buffer;
    source.connect(this.audioContext.destination);

    const now = this.audioContext.currentTime;
    const startTime = Math.max(now, this.nextStartTime);
    source.start(startTime);
    this.nextStartTime = startTime + buffer.duration;
  }

  // Recreating the context immediately silences anything already
  // scheduled/playing - simpler and more reliable than tracking every
  // individual buffer source node to stop it.
  stopAll() {
    this.audioContext.close();
    this.audioContext = new AudioContext({ sampleRate: PLAYBACK_SAMPLE_RATE });
    this.nextStartTime = 0;
  }

  close() {
    this.audioContext.close();
  }
}
