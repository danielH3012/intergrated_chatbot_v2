import { useEffect, useRef, useState } from 'react'
import { apiFetch, getToken } from './api.js'
import { useAuth } from './AuthContext.jsx'
import { VoiceRecorder } from './audioRecorder.js'

function formatTime(date) {
  if (!date || isNaN(new Date(date).getTime())) return ''
  return new Date(date).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function formatDuration(sec) {
  const m = Math.floor(sec / 60).toString().padStart(2, '0')
  const s = (sec % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

function formatFileSize(bytes) {
  if (!bytes || bytes <= 0) return ''
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function getFileIcon(filename = '', mimeType = '') {
  const ext = (filename.split('.').pop() || '').toLowerCase()
  if (ext === 'pdf' || (mimeType && mimeType.includes('pdf'))) return '📄'
  if (['png', 'jpg', 'jpeg', 'gif', 'webp'].includes(ext) || (mimeType && mimeType.includes('image'))) return '🖼️'
  if (['csv', 'xlsx', 'xls'].includes(ext) || (mimeType && (mimeType.includes('spreadsheet') || mimeType.includes('csv')))) return '📊'
  if (['doc', 'docx', 'txt', 'rtf'].includes(ext)) return '📝'
  return '📎'
}

export default function Chat() {
  const { user } = useAuth()
  const chatID = user?.username || 'demo'
  const userRole = user?.role || 'viewer'

  const [messages, setMessages] = useState([])
  const [input, setInput] = useState('')
  const [model, setModel] = useState('ox-alpha-free')
  const [connected, setConnected] = useState(false)
  const [isRecording, setIsRecording] = useState(false)
  const [recordSeconds, setRecordSeconds] = useState(0)
  const [isTranscribing, setIsTranscribing] = useState(false)
  const [attachment, setAttachment] = useState(null)
  const [playingMsgId, setPlayingMsgId] = useState(null)
  const [loadingTtsId, setLoadingTtsId] = useState(null)

  const wsRef = useRef(null)
  const bottomRef = useRef(null)
  const inputRef = useRef(null)
  const fileInputRef = useRef(null)
  const lastSentRef = useRef({ text: '', at: 0 })
  const recorderRef = useRef(null)
  const timerRef = useRef(null)
  const audioPlayerRef = useRef(null)

  const stopAllAudio = () => {
    if (audioPlayerRef.current) {
      audioPlayerRef.current.pause()
      audioPlayerRef.current.currentTime = 0
      audioPlayerRef.current = null
    }
    setPlayingMsgId(null)
    setLoadingTtsId(null)
  }

  const toggleSpeak = async (id, text) => {
    if (playingMsgId === id || loadingTtsId === id) {
      stopAllAudio()
      return
    }

    if (!text || !text.trim()) return

    stopAllAudio()
    setLoadingTtsId(id)

    try {
      const resp = await apiFetch('/tts', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text }),
      })

      if (resp.ok) {
        const blob = await resp.blob()
        const audioUrl = URL.createObjectURL(blob)
        const audio = new Audio(audioUrl)
        audioPlayerRef.current = audio

        audio.onended = () => {
          setPlayingMsgId(null)
          setLoadingTtsId(null)
          URL.revokeObjectURL(audioUrl)
        }
        audio.onerror = () => {
          setPlayingMsgId(null)
          setLoadingTtsId(null)
          URL.revokeObjectURL(audioUrl)
        }

        setLoadingTtsId(null)
        setPlayingMsgId(id)
        await audio.play()
        return
      } else {
        const errJson = await resp.json().catch(() => ({}))
        console.error('[TTS] Backend error:', resp.status, errJson)
      }
    } catch (err) {
      console.error('[TTS] Failed to fetch audio from backend:', err)
    } finally {
      setLoadingTtsId(null)
    }
  }

  useEffect(() => {
    return () => {
      stopAllAudio()
    }
  }, [])

  useEffect(() => {
    let closed = false
    let retry = null

    const connect = () => {
      const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
      const token = getToken()
      const tokenQuery = token ? `?token=${encodeURIComponent(token)}` : ''
      const wsUrl = `${proto}://${window.location.host}/ws/${chatID}${tokenQuery}`

      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => setConnected(true)
      ws.onclose = () => {
        setConnected(false)
        if (!closed) retry = setTimeout(connect, 1500)
      }
      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          const last = lastSentRef.current
          const isSelfEcho =
            data.user === true &&
            (data.text === last.displayText || data.text === last.rawText) &&
            Date.now() - last.at < 3000

          if (isSelfEcho) return
          const attachInfo = data.attachment_info || (data.attachment ? { name: data.attachment } : null)
          setMessages((prev) => {
            // Deduplicate by message ID to prevent double bubbles
            if (data.id && prev.some((m) => m.id === data.id)) return prev
            return [
              ...prev,
              {
                id: data.id ?? `${Date.now()}-${Math.random()}`,
                user: !!data.user,
                username: data.username || (data.user ? user?.username || 'You' : 'QTERA AI'),
                role: data.role || (data.user ? userRole : 'system'),
                text: data.text,
                attachment: attachInfo,
                time: new Date(),
              },
            ]
          })
        } catch {
          // ignore malformed frames
        }
      }
    }

    connect()
    return () => {
      closed = true
      clearTimeout(retry)
      wsRef.current?.close()
    }
  }, [chatID, user, userRole])

  useEffect(() => {
    apiFetch(`/messages/${chatID}`)
      .then((res) => (res.ok ? res.json() : []))
      .then((history) =>
        setMessages(
          history.map((m) => {
            let attach = m.attachment || null
            let text = m.chat || ''

            // Fallback for legacy messages that stored 📎 [Attached: filename] in text
            if (!attach && text.includes('📎 [Attached: ')) {
              const match = text.match(/📎 \[Attached:\s*([^\]]+)\]/)
              if (match) {
                attach = { name: match[1].trim() }
              }
            }

            return {
              id: m.id ?? `${Date.now()}-${Math.random()}`,
              user: !!m.user,
              username: m.username || (m.user ? user?.username || 'You' : 'QTERA AI'),
              role: m.role || (m.user ? userRole : 'system'),
              text,
              attachment: attach,
              time: new Date(m.created_at),
            }
          })
        )
      )
      .catch(() => {})
  }, [chatID, user, userRole])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const send = () => {
    const text = input.trim()
    if (!text && !attachment) return

    const time = new Date()
    const fileToSend = attachment
    const displayText = fileToSend
      ? `${text ? text + '\n' : ''}📎 [Attached: ${fileToSend.name}]`
      : text

    const clientMsgId = crypto.randomUUID()
    const optimisticAttach = fileToSend
      ? {
          name: fileToSend.name,
          size: fileToSend.size,
          type: fileToSend.type,
          url: URL.createObjectURL(fileToSend),
        }
      : null

    setMessages((prev) => [
      ...prev,
      {
        id: clientMsgId,
        user: true,
        username: user?.username || 'You',
        role: userRole,
        text: displayText,
        attachment: optimisticAttach,
        time,
      },
    ])
    setInput('')
    setAttachment(null)
    if (fileInputRef.current) fileInputRef.current.value = ''
    const rawText = fileToSend
      ? text || `Process attached file: ${fileToSend.name}`
      : text
    lastSentRef.current = { displayText, rawText, at: Date.now() }

    if (fileToSend) {
      const formData = new FormData()
      formData.append('chat_id', chatID)
      formData.append('id', clientMsgId)
      formData.append('text', rawText)
      formData.append('models', model)
      formData.append('attachment', fileToSend)

      const token = getToken()
      const headers = {}
      if (token) headers['Authorization'] = `Bearer ${token}`

      fetch('/message', {
        method: 'POST',
        headers,
        body: formData,
      }).catch(() => {})
    } else {
      apiFetch('/message', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ chat_id: chatID, text, models: model }),
      }).catch(() => {})
    }
  }

  const toggleRecording = async () => {
    if (isRecording) {
      clearInterval(timerRef.current)
      setIsRecording(false)
      setIsTranscribing(true)

      try {
        const wavBlob = await recorderRef.current.stop()
        recorderRef.current = null

        if (!wavBlob || wavBlob.size === 0) {
          alert('No audio recorded. Please ensure microphone permissions are granted.')
          return
        }

        const formData = new FormData()
        formData.append('audio', wavBlob, 'recording.wav')

        const token = getToken()
        const headers = {}
        if (token) headers['Authorization'] = `Bearer ${token}`

        const res = await fetch('/transcribe', {
          method: 'POST',
          headers,
          body: formData,
        })
        if (res.ok) {
          const data = await res.json()
          const text = (data && data.text ? data.text : '').trim()
          if (text) {
            setInput((prev) => (prev ? prev + ' ' + text : text))
            setTimeout(() => inputRef.current?.focus(), 50)
          } else {
            alert('No speech was detected in your recording. Please speak clearly into your microphone.')
          }
        } else {
          const errData = await res.json().catch(() => ({}))
          console.warn('Transcription failed:', errData)
          alert(errData.details || errData.error || 'Failed to transcribe audio. Please ensure whisper-cli and models are installed.')
        }
      } catch (err) {
        console.error('Error during voice transcription:', err)
        alert('Voice transcription encountered an error: ' + (err.message || err))
      } finally {
        setIsTranscribing(false)
      }
    } else {
      try {
        const recorder = new VoiceRecorder()
        await recorder.start()
        recorderRef.current = recorder
        setIsRecording(true)
        setRecordSeconds(0)
        timerRef.current = setInterval(() => {
          setRecordSeconds((sec) => sec + 1)
        }, 1000)
      } catch (err) {
        console.error('Failed to start microphone:', err)
        alert('Could not access microphone. Please ensure microphone permissions are granted.')
      }
    }
  }

  const getRoleBadgeClass = (r) => {
    const norm = (r || '').toLowerCase()
    if (norm === 'admin') return 'badge-admin'
    if (norm === 'operator') return 'badge-operator'
    return 'badge-viewer'
  }

  return (
    <div className="chat-container">
      <div className="chat-app">
        <header className="chat-header">
          <div className="chat-header-main">
            <div className="chat-avatar">Q</div>
            <div className="chat-header-info">
              <div className="chat-title-row">
                <span className="chat-name">QTERA Security AI</span>
                <span className={`chat-conn-status ${connected ? 'online' : 'connecting'}`}>
                  {connected ? 'connected' : 'connecting…'}
                </span>
              </div>
              <div className="chat-meta-row">
                <span>Logged in as: <strong>{user?.username || 'Guest'}</strong></span>
                <span className={`chat-role-pill ${getRoleBadgeClass(userRole)}`}>
                  {userRole.toUpperCase()}
                </span>
              </div>
            </div>
          </div>

          <div className="role-capability-banner">
            {userRole === 'admin' && (
              <span>⚡ <strong>Admin Mode</strong>: Full tool access (Agents, Telemetry, Hardware, Add Groups).</span>
            )}
            {userRole === 'operator' && (
              <span>🛠️ <strong>Operator Mode</strong>: Telemetry &amp; agent inspection enabled (Group write locked).</span>
            )}
            {userRole === 'viewer' && (
              <span>👁️ <strong>Viewer Mode</strong>: Read-only knowledge and group overview access.</span>
            )}
          </div>
        </header>

        <main className="chat-messages">
          {messages.length === 0 && (
            <div className="chat-empty-state">
              <div className="empty-icon">💬</div>
              <h3>How can I assist you with your SIEM data today?</h3>
              <p>Ask about active agents, hardware metrics, security events, or group memberships.</p>
            </div>
          )}

          {messages.map((msg) => (
            <div key={msg.id} className={`chat-msg ${msg.user ? 'right' : 'left'}`}>
              <div className="chat-msg-header">
                <span className="msg-sender">{msg.username}</span>
                {msg.role && (
                  <span className={`msg-role-tag ${getRoleBadgeClass(msg.role)}`}>
                    {msg.role}
                  </span>
                )}
              </div>
              <div className="chat-bubble">
                {(() => {
                  const cleanText = msg.text
                    ? msg.text.replace(/\n?📎 \[Attached: [^\]]+\]/, '').trim()
                    : ''
                  return cleanText ? (
                    <div className="chat-text">{cleanText}</div>
                  ) : !msg.user ? (
                    <div className="chat-text" style={{ fontStyle: 'italic', opacity: 0.7 }}>
                      I do not have information or unable to do that.
                    </div>
                  ) : null
                })()}

                {msg.attachment && (
                  <div className="chat-bubble-attachment">
                    <div className="bubble-attachment-icon">
                      {getFileIcon(msg.attachment.name, msg.attachment.type)}
                    </div>
                    <div className="bubble-attachment-info">
                      {msg.attachment.url ? (
                        <a
                          href={msg.attachment.url}
                          target="_blank"
                          rel="noreferrer"
                          download={msg.attachment.name || true}
                          className="bubble-attachment-name"
                          style={{ textDecoration: 'none', color: 'inherit', cursor: 'pointer' }}
                          title={`Open or download ${msg.attachment.name}`}
                        >
                          {msg.attachment.name}
                        </a>
                      ) : (
                        <div className="bubble-attachment-name" title={msg.attachment.name}>
                          {msg.attachment.name}
                        </div>
                      )}
                      {msg.attachment.size > 0 && (
                        <div className="bubble-attachment-size">
                          {formatFileSize(msg.attachment.size)}
                        </div>
                      )}
                    </div>
                    {msg.attachment.url && (
                      <a
                        href={msg.attachment.url}
                        target="_blank"
                        rel="noreferrer"
                        className="bubble-attachment-download"
                        download={msg.attachment.name || true}
                        title={`Download ${msg.attachment.name}`}
                      >
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                          <polyline points="7 10 12 15 17 10" />
                          <line x1="12" y1="15" x2="12" y2="3" />
                        </svg>
                      </a>
                    )}
                  </div>
                )}
                <div className="chat-bubble-footer">
                  {!msg.user && (
                    <button
                      type="button"
                      className={`chat-tts-btn ${playingMsgId === msg.id ? 'playing' : ''} ${loadingTtsId === msg.id ? 'loading' : ''}`}
                      onClick={() => toggleSpeak(msg.id, msg.text)}
                      title={
                        loadingTtsId === msg.id
                          ? 'Generating voice with Cartesia Sonic 3.6...'
                          : playingMsgId === msg.id
                          ? 'Stop audio'
                          : 'Listen with Cartesia Sonic 3.6'
                      }
                      disabled={loadingTtsId === msg.id}
                    >
                      {loadingTtsId === msg.id ? (
                        <span className="tts-spinner" />
                      ) : playingMsgId === msg.id ? (
                        <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
                          <rect x="6" y="6" width="12" height="12" rx="2" />
                        </svg>
                      ) : (
                        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                          <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5" />
                          <path d="M15.54 8.46a5 5 0 0 1 0 7.07" />
                          <path d="M19.07 4.93a10 10 0 0 1 0 14.14" />
                        </svg>
                      )}
                    </button>
                  )}
                  <span className="chat-time">{formatTime(msg.time)}</span>
                </div>
              </div>
            </div>
          ))}
          <div ref={bottomRef} />
        </main>

        <footer className="chat-composer-wrap">
          {attachment && (
            <div className="chat-attachment-pill">
              <span className="attachment-icon">📎</span>
              <span className="attachment-name">{attachment.name}</span>
              <button
                type="button"
                className="attachment-remove-btn"
                onClick={() => {
                  setAttachment(null)
                  if (fileInputRef.current) fileInputRef.current.value = ''
                }}
                title="Remove attachment"
              >
                ✕
              </button>
            </div>
          )}

          <div className="chat-composer">
            <input
              type="file"
              ref={fileInputRef}
              style={{ display: 'none' }}
              accept=".pdf,.png,.jpg,.jpeg,.txt,.csv,.xlsx,.xls"
              onChange={(e) => {
                if (e.target.files && e.target.files[0]) {
                  setAttachment(e.target.files[0])
                }
              }}
            />

            <button
              type="button"
              className="chat-attach-btn"
              onClick={() => fileInputRef.current?.click()}
              disabled={isRecording || isTranscribing}
              title="Attach document, spreadsheet, or image (PDF, PNG, JPG, TXT, CSV, Excel)"
            >
              <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48" />
              </svg>
            </button>

            {isRecording ? (
              <div className="recording-status-pill" style={{ flex: 1 }}>
                <div className="recording-live-dot" />
                <span>Recording audio... {formatDuration(recordSeconds)} (Click mic to stop)</span>
              </div>
            ) : isTranscribing ? (
              <div className="recording-status-pill" style={{ flex: 1, color: '#fde047', borderColor: '#eab308' }}>
                <span>⏳ Transcribing speech with Whisper...</span>
              </div>
            ) : (
              <input
                ref={inputRef}
                className="chat-input"
                type="text"
                value={input}
                placeholder={attachment ? `Ask about ${attachment.name}…` : `Ask QTERA AI as ${user?.username || 'user'} (${userRole})…`}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && send()}
              />
            )}

          <select
            className="chat-model-select"
            value={model}
            onChange={(e) => setModel(e.target.value)}
            title="Select AI Model"
          >
            <option value="ox-alpha-free">OX Alpha Free</option>
            <option value="glm-5.3-flash">GLM 5.3 Flash</option>
            <option value="mimo-v2.5">MiMo V2.5</option>
            <option value="gpt-5.6-luna">GPT-5.6 Luna</option>
            <option value="qwen3.8-flash">Qwen 3.8 Flash</option>
            <option value="deepseek-v4-flash">DeepSeek V4 Flash</option>
            <option value="liquid/lfm-2.5-2.6b:free">LFM 2.5 (Free)</option>
            <option value="z-ai/glm-5.2:free">GLM 5.2 (Free)</option>
          </select>

          <button
            type="button"
            className={`chat-mic-btn ${isRecording ? 'recording' : ''} ${isTranscribing ? 'transcribing' : ''}`}
            onClick={toggleRecording}
            disabled={isTranscribing}
            title={isRecording ? 'Stop recording & transcribe' : 'Record voice message'}
          >
            {isRecording ? (
              <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                <rect x="6" y="6" width="12" height="12" rx="2" />
              </svg>
            ) : isTranscribing ? (
              <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="12" cy="12" r="10" strokeDasharray="32" strokeDashoffset="16" />
              </svg>
            ) : (
              <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" />
                <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
                <line x1="12" y1="19" x2="12" y2="23" />
                <line x1="8" y1="23" x2="16" y2="23" />
              </svg>
            )}
          </button>

            <button className="chat-send-btn" onClick={send} disabled={isRecording || isTranscribing} aria-label="Send message">
              <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M22 2L11 13" />
                <path d="M22 2l-7 20-4-9-9-4 20-7z" />
              </svg>
            </button>
          </div>
        </footer>
      </div>
    </div>
  )
}
