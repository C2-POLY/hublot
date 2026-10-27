// SPDX-License-Identifier: AGPL-3.0-or-later
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import type { PortalRequest } from '../api/client'
import { formatDay } from '../lib/format'
import { timelineIndex } from '../lib/status'
import { PrivateBadge, StatusPill } from './StatusPill'

/** One request in a list: project colour stripe, title, meta, (progress), status. */
export function RequestRow({ request: r, showProgress = false }: { request: PortalRequest; showProgress?: boolean }) {
  const { t, i18n } = useTranslation()
  const date = formatDay(r.createdAt, i18n.language)
  const step = timelineIndex(r.status) + 1
  return (
    <Link
      to={`/demandes/${r.id}`}
      className={`grid items-stretch overflow-hidden rounded-2xl border-franc bg-surface text-ink no-underline hover:text-ink ${
        showProgress ? 'grid-cols-[12px_minmax(0,1fr)_auto] md:grid-cols-[12px_minmax(0,1fr)_200px_140px]' : 'grid-cols-[12px_minmax(0,1fr)_auto]'
      }`}
    >
      <span aria-hidden="true" className="border-r-[2.5px] border-ink" style={{ background: r.project.color }} />
      <span className="flex min-w-0 flex-col gap-1 px-4 py-3.5 sm:px-[18px]">
        <span className="flex flex-wrap items-center gap-2">
          <strong className="text-[17px] sm:text-lg">{r.title}</strong>
          {r.private && <PrivateBadge />}
        </span>
        <span className="text-sm text-ink-2">
          {t('requests.meta', { project: r.project.name, kind: t(`kinds.${r.kind}.lower`), date })}
        </span>
      </span>
      {showProgress && (
        <span className="hidden items-center py-4 md:flex">
          {step > 0 && (
            <span role="img" aria-label={t('requests.progress', { step })} className="flex w-full gap-1">
              {[1, 2, 3, 4, 5].map((n) => (
                <span
                  key={n}
                  className={`h-2.5 flex-1 rounded-[5px] border-2 border-ink ${n < step || r.status === 'available' ? 'bg-ink' : n === step ? 'bg-accent' : ''}`}
                />
              ))}
            </span>
          )}
        </span>
      )}
      <span className="flex items-center justify-end px-4 py-3.5 sm:px-[18px]">
        <StatusPill status={r.status} />
      </span>
    </Link>
  )
}
