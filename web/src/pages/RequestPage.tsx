// SPDX-License-Identifier: AGPL-3.0-or-later
import { Link, useParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import { ApiError, type PortalRequest } from '../api/client'
import { useRequest } from '../api/hooks'
import { VehicleEmblem } from '../components/Emblem'
import { BoxIcon, CheckIcon, ClockIcon } from '../components/Icons'
import { AttachmentTile } from '../components/AttachmentTile'
import { ErrorState, Loading } from '../components/States'
import { PrivateBadge } from '../components/StatusPill'
import { formatDay } from '../lib/format'
import { TIMELINE, timelineIndex } from '../lib/status'

export function RequestPage() {
  const { t } = useTranslation()
  const { id = '' } = useParams()
  const request = useRequest(id)

  if (request.isPending) return <Loading />
  if (request.isError) {
    if (request.error instanceof ApiError && request.error.status === 404) {
      return (
        <section className="flex max-w-2xl flex-col items-start gap-5 py-10">
          <h1 className="m-0 text-[44px] leading-[0.95] tracking-[-0.03em]">{t('track.notFoundTitle')}</h1>
          <p className="m-0 text-lg">{t('track.notFoundText')}</p>
          <Link to="/demandes" className="btn-primary h-14 px-6 text-lg">
            {t('track.backToList')}
          </Link>
        </section>
      )
    }
    return <ErrorState message={t('common.error')} onRetry={() => void request.refetch()} />
  }
  return <RequestView r={request.data} />
}

function RequestView({ r }: { r: PortalRequest }) {
  const { t, i18n } = useTranslation()
  const current = timelineIndex(r.status)
  // Answered or closed requests sit outside the parcel steps: everything before stays done.
  const reached = current >= 0 ? current : 0
  const offTimeline = current < 0

  return (
    <div className="flex flex-col gap-7">
      <section
        className="rounded-[22px] border-franc-strong px-5 py-6 shadow-hard-lg sm:px-7"
        style={{ background: r.project.color }}
      >
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <p className="m-0 mb-2 flex flex-wrap items-center gap-2 text-[15px] font-bold">
              {t('track.meta', {
                project: r.project.name,
                kind: t(`kinds.${r.kind}.lower`),
                date: formatDay(r.createdAt, i18n.language, true),
              })}
              {r.private && <PrivateBadge />}
            </p>
            <h1 className="m-0 break-words text-[34px] leading-none tracking-[-0.03em] sm:text-[48px]">{r.title}</h1>
          </div>
          <span className="hidden sm:block">
            <VehicleEmblem vehicle={r.project.vehicle} color={r.project.color} size={88} />
          </span>
        </div>

        <ol aria-label={t('track.timeline')} className="m-0 mt-7 grid list-none grid-cols-1 gap-2.5 p-0 sm:grid-cols-5">
          {TIMELINE.map((step, i) => {
            const done = i < reached || (offTimeline && i === 0) || (step === 'available' && current === i)
            const isCurrent = !offTimeline && i === current && step !== 'available'
            const label = t(`status.${step}`)
            if (done) {
              return (
                <li key={step} className="flex flex-row items-center gap-2 rounded-[14px] border-franc bg-ink p-3.5 text-white sm:flex-col sm:items-start sm:gap-1.5">
                  <CheckIcon color="#7FD8B6" />
                  <span className="text-[17px] font-bold">{label}</span>
                  {i === 0 && (
                    <span className="text-sm text-[#D6D6D6]">{t('track.receivedOn', { date: formatDay(r.createdAt, i18n.language) })}</span>
                  )}
                </li>
              )
            }
            if (isCurrent) {
              return (
                <li
                  key={step}
                  aria-current="step"
                  className="flex flex-row items-center gap-2 rounded-[14px] border-franc-strong bg-accent p-3.5 shadow-hard sm:-translate-y-1 sm:flex-col sm:items-start sm:gap-1.5"
                >
                  <ClockIcon />
                  <span className="text-[17px] font-bold">{label}</span>
                  {i === 0 && <span className="text-sm">{t('track.receivedOn', { date: formatDay(r.createdAt, i18n.language) })}</span>}
                </li>
              )
            }
            return (
              <li key={step} className="flex flex-row items-center gap-2 rounded-[14px] border-franc-dashed bg-surface p-3.5 sm:flex-col sm:items-start sm:gap-1.5">
                <BoxIcon />
                <span className="text-[17px] font-bold">{label}</span>
              </li>
            )
          })}
        </ol>
        <p className="m-0 mt-[22px] text-[19px] font-bold leading-snug">{t(`statusLine.${r.status}`)}</p>
      </section>

      <div className="flex flex-wrap items-start gap-7">
        <section aria-labelledby="track-talk" className="flex min-w-0 flex-[999_1_560px] flex-col gap-3.5">
          <h2 id="track-talk" className="m-0 text-[26px]">
            {t('track.talk')}
          </h2>
          <p className="m-0 rounded-2xl border-franc-dashed bg-surface p-[18px] text-[17px]">{t('track.talkSoon')}</p>

          <h2 id="track-files" className="m-0 mt-3 text-[26px]">
            {t('track.attachments')}
          </h2>
          {r.attachments.length === 0 ? (
            <p className="m-0 text-ink-2">{t('track.noAttachments')}</p>
          ) : (
            <ul aria-labelledby="track-files" className="m-0 grid list-none grid-cols-2 gap-3 p-0 sm:grid-cols-4">
              {r.attachments.map((a) => (
                <li key={a.id}>
                  <AttachmentTile attachment={a} />
                </li>
              ))}
            </ul>
          )}
        </section>
        <aside className="flex min-w-0 flex-[1_1_280px] flex-col gap-3.5">
          <p className="m-0 rounded-2xl border-franc bg-surface p-[18px] text-sm text-ink-2">
            {t('track.reference', { number: r.number })}
          </p>
        </aside>
      </div>
    </div>
  )
}
