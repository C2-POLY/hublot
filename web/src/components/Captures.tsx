// SPDX-License-Identifier: AGPL-3.0-or-later
// Screenshots without effort: paste, drag and drop, file picker, or capture the screen.
import { useEffect, useRef, useState, type DragEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { api, ApiError, type Attachment } from '../api/client'
import { AttachmentTile } from './AttachmentTile'

const ACCEPT = 'image/png,image/jpeg,image/gif,image/webp,application/pdf,text/plain'

function canCaptureScreen(): boolean {
  return typeof navigator !== 'undefined' && typeof navigator.mediaDevices?.getDisplayMedia === 'function'
}

async function grabScreen(fileName: string): Promise<File | null> {
  let stream: MediaStream
  try {
    stream = await navigator.mediaDevices.getDisplayMedia({ video: true, audio: false })
  } catch (err) {
    // The person closed the browser's picker: not an error worth showing.
    if (err instanceof DOMException && err.name === 'NotAllowedError') return null
    throw err
  }
  try {
    const video = document.createElement('video')
    video.srcObject = stream
    video.muted = true
    await video.play()
    await new Promise((r) => requestAnimationFrame(() => r(null)))
    const canvas = document.createElement('canvas')
    canvas.width = video.videoWidth
    canvas.height = video.videoHeight
    canvas.getContext('2d')?.drawImage(video, 0, 0)
    const blob = await new Promise<Blob | null>((r) => canvas.toBlob(r, 'image/png'))
    return blob ? new File([blob], fileName, { type: 'image/png' }) : null
  } finally {
    stream.getTracks().forEach((t) => t.stop())
  }
}

export function Captures({
  attachments,
  onAdd,
  onRemove,
}: {
  attachments: Attachment[]
  onAdd: (a: Attachment) => void
  onRemove: (id: string) => void
}) {
  const { t } = useTranslation()
  const inputRef = useRef<HTMLInputElement>(null)
  const [pending, setPending] = useState<string[]>([])
  const [messages, setMessages] = useState<string[]>([])
  const [dragging, setDragging] = useState(false)

  const upload = async (files: File[]) => {
    setMessages([])
    for (const file of files) {
      setPending((p) => [...p, file.name])
      try {
        onAdd(await api.uploadAttachment(file))
      } catch (err) {
        const key =
          err instanceof ApiError && err.status === 413
            ? 'wizard.tooLarge'
            : err instanceof ApiError && err.status === 415
              ? 'wizard.badType'
              : 'wizard.uploadFailed'
        setMessages((m) => [...m, t(key, { name: file.name })])
      } finally {
        setPending((p) => {
          const i = p.indexOf(file.name)
          return i < 0 ? p : [...p.slice(0, i), ...p.slice(i + 1)]
        })
      }
    }
  }

  // Ctrl+V anywhere on the step: only files are taken, pasted text goes where it belongs.
  const uploadRef = useRef(upload)
  uploadRef.current = upload
  useEffect(() => {
    const onPaste = (e: ClipboardEvent) => {
      const files = Array.from(e.clipboardData?.files ?? [])
      if (files.length === 0) return
      e.preventDefault()
      void uploadRef.current(files)
    }
    window.addEventListener('paste', onPaste)
    return () => window.removeEventListener('paste', onPaste)
  }, [])

  const onDrop = (e: DragEvent) => {
    e.preventDefault()
    setDragging(false)
    const files = Array.from(e.dataTransfer.files)
    if (files.length) void upload(files)
  }

  const capture = async () => {
    try {
      const file = await grabScreen(t('wizard.screenshotName'))
      if (file) await upload([file])
    } catch {
      setMessages([t('wizard.captureFailed')])
    }
  }

  const remove = async (id: string) => {
    try {
      await api.deleteAttachment(id)
    } catch (err) {
      if (!(err instanceof ApiError && err.status === 404)) {
        setMessages([t('common.error')])
        return
      }
    }
    onRemove(id)
  }

  return (
    <div className="flex flex-col gap-2.5">
      <p className="m-0 text-[15px] font-bold">{t('wizard.captures')}</p>
      {attachments.length > 0 && (
        <ul className="m-0 grid list-none grid-cols-2 gap-3 p-0">
          {attachments.map((a) => (
            <li key={a.id}>
              <AttachmentTile attachment={a} onRemove={() => void remove(a.id)} />
            </li>
          ))}
        </ul>
      )}
      <div className={`grid gap-2 ${canCaptureScreen() ? 'grid-cols-2' : 'grid-cols-1'}`}>
        {canCaptureScreen() && (
          <button type="button" onClick={() => void capture()} className="btn-light h-11 rounded-xl text-sm">
            {t('wizard.captureScreen')}
          </button>
        )}
        <button type="button" onClick={() => inputRef.current?.click()} className="btn-light h-11 rounded-xl text-sm">
          {t('wizard.addFile')}
        </button>
        <input
          ref={inputRef}
          type="file"
          accept={ACCEPT}
          multiple
          hidden
          onChange={(e) => {
            const files = Array.from(e.target.files ?? [])
            e.target.value = ''
            if (files.length) void upload(files)
          }}
        />
      </div>
      <div
        onDragOver={(e) => {
          e.preventDefault()
          setDragging(true)
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
        className={`rounded-xl border-2 border-dashed border-ink p-3 text-center text-sm ${dragging ? 'bg-notice' : ''}`}
      >
        {t('wizard.dropHint')}
      </div>
      <div role="status" aria-live="polite" className="flex flex-col gap-1 text-sm">
        {pending.map((name, i) => (
          <span key={`${name}-${i}`}>{t('wizard.uploading', { name })}</span>
        ))}
        {messages.map((m, i) => (
          <span key={i} className="font-bold">
            {m}
          </span>
        ))}
      </div>
    </div>
  )
}
