// SPDX-License-Identifier: AGPL-3.0-or-later
import { useTranslation } from 'react-i18next'
import type { Attachment } from '../api/client'
import { formatSize } from '../lib/format'
import { CloseIcon, FileIcon } from './Icons'

/** A thumbnail for images, a file card otherwise. Opens the file in a new tab. */
export function AttachmentTile({ attachment: a, onRemove }: { attachment: Attachment; onRemove?: () => void }) {
  const { t, i18n } = useTranslation()
  const isImage = a.contentType.startsWith('image/')
  return (
    <div className="relative">
      <a
        href={a.url}
        target="_blank"
        rel="noopener"
        className="flex flex-col overflow-hidden rounded-xl border-franc bg-surface text-ink no-underline hover:text-ink"
      >
        {isImage ? (
          <img src={a.url} alt={a.name} loading="lazy" className="block h-24 w-full border-b-2 border-ink object-cover" />
        ) : (
          <span className="flex h-24 items-center justify-center border-b-2 border-ink bg-soft">
            <FileIcon />
          </span>
        )}
        <span className="flex flex-col px-2.5 py-1.5 text-xs">
          <strong className="truncate">{a.name}</strong>
          <span className="text-ink-2">{formatSize(a.size, i18n.language)}</span>
        </span>
      </a>
      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          aria-label={t('wizard.remove', { name: a.name })}
          className="absolute -right-2 -top-2 flex h-8 w-8 items-center justify-center rounded-full border-franc bg-surface"
        >
          <CloseIcon />
        </button>
      )}
    </div>
  )
}
