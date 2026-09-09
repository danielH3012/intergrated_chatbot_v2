/**
 * VoiceRecorder — records microphone audio and returns a WAV Blob on stop().
 *
 * Usage:
 *   const recorder = new VoiceRecorder()
 *   await recorder.start()
 *   const wavBlob = await recorder.stop()
 */
export class VoiceRecorder {
  constructor() {
    this._mediaRecorder = null
    this._chunks = []
    this._stream = null
    this._resolveStop = null
  }

  /**
   * Request microphone access and start recording.
   * @throws if the user denies microphone permission.
   */
  async start() {
    this._chunks = []
    this._stream = await navigator.mediaDevices.getUserMedia({ audio: true })

    // Prefer audio/webm; fall back to the browser default
    const mimeType = MediaRecorder.isTypeSupported('audio/webm;codecs=opus')
      ? 'audio/webm;codecs=opus'
      : ''

    this._mediaRecorder = new MediaRecorder(this._stream, mimeType ? { mimeType } : undefined)

    this._mediaRecorder.ondataavailable = (e) => {
      if (e.data && e.data.size > 0) {
        this._chunks.push(e.data)
      }
    }

    this._mediaRecorder.onstop = () => {
      const blob = new Blob(this._chunks, {
        type: this._mediaRecorder.mimeType || 'audio/webm',
      })
      this._cleanup()
      if (this._resolveStop) {
        this._resolveStop(blob)
        this._resolveStop = null
      }
    }

    this._mediaRecorder.start(250) // collect data every 250 ms
  }

  /**
   * Stop recording and return the audio Blob.
   * @returns {Promise<Blob>}
   */
  stop() {
    return new Promise((resolve, reject) => {
      if (!this._mediaRecorder) {
        reject(new Error('Recorder not started'))
        return
      }
      this._resolveStop = resolve
      this._mediaRecorder.stop()
    })
  }

  _cleanup() {
    if (this._stream) {
      this._stream.getTracks().forEach((t) => t.stop())
      this._stream = null
    }
  }
}
