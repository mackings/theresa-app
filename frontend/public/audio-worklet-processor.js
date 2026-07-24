// Plain JS, not TypeScript: AudioWorklet.addModule() fetches this as a URL
// and runs it in the audio rendering thread - it does not go through
// Next.js's bundler like a normal import would.
class PCMCaptureProcessor extends AudioWorkletProcessor {
  constructor() {
    super();
    this.chunkSamples = 1600; // ~100ms at 16kHz
    this.buffer = new Int16Array(this.chunkSamples);
    this.offset = 0;
  }

  process(inputs) {
    const input = inputs[0];
    if (!input || input.length === 0) {
      return true;
    }

    const channel = input[0];
    for (let i = 0; i < channel.length; i++) {
      const clamped = Math.max(-1, Math.min(1, channel[i]));
      this.buffer[this.offset] = clamped < 0 ? clamped * 0x8000 : clamped * 0x7fff;
      this.offset++;

      if (this.offset >= this.chunkSamples) {
        this.port.postMessage(this.buffer.buffer.slice(0));
        this.offset = 0;
      }
    }

    return true;
  }
}

registerProcessor("pcm-capture-processor", PCMCaptureProcessor);
