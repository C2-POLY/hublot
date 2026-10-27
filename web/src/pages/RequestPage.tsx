// SPDX-License-Identifier: AGPL-3.0-or-later
import { Link, useParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import { ApiError, type PortalRequest } from '../api/client'
import { useRequest } from '../api/hooks'
import { VehicleEmblem } from '../components/Emblem'
import { BoxIcon, CheckIcon, ClockIcon, CloseIcon } from '../components/Icons'
import { AttachmentTile } from '../components/AttachmentTile'
import { ErrorState, Loading } from '../components/States'
import { PrivateBadge } from '../components/StatusPill'
import { formatDay } from '../lib/format'
import { parcelSteps, statusSentenceKey, type Step } from '../lib/status'

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
  const steps = parcelSteps(r)

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

        <ol
          aria-label={t('track.timeline')}
          className="m-0 mt-6 flex list-none flex-col p-0 sm:mt-7 sm:grid sm:auto-cols-fr sm:grid-flow-col sm:gap-2.5"
        >
          {steps.map((step) => (
            <TimelineStep key={step.status} step={step} milestone={r.milestone} />
          ))}
        </ol>
        <p className="m-0 mt-[22px] text-[19px] font-bold leading-snug">
          {t(statusSentenceKey(r), { milestone: r.milestone })}
        </p>
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

// One step of the parcel timeline: a compact row on phones (C-SuiviMobile), a tile from sm up (C-Suivi).
function TimelineStep({ step, milestone }: { step: Step; milestone?: string }) {
  const { t, i18n } = useTranslation()
  const ending = step.status === 'answered' || step.status === 'closed'
  const date = step.at ? formatDay(step.at, i18n.language) : ''
  const label = t(`status.${step.status}`)
  const row = 'grid grid-cols-[30px_minmax(0,1fr)_auto] items-center gap-3 sm:flex sm:flex-col sm:items-start sm:gap-1.5 sm:rounded-[14px] sm:p-3.5'
  const mark = 'flex h-[26px] w-[26px] items-center justify-center rounded-lg sm:h-auto sm:w-auto sm:rounded-none sm:border-0 sm:bg-transparent'

  if (step.state === 'current' || ending) {
    const bg = step.status === 'answered' ? 'bg-mint' : step.status === 'closed' ? 'bg-surface' : 'bg-accent'
    return (
      <li
        aria-current="step"
        className={`${row} my-1 -mx-2.5 min-h-[52px] rounded-[14px] border-franc-strong px-2.5 shadow-hard sm:m-0 sm:-translate-y-1 ${bg}`}
      >
        <span aria-hidden="true" className={`${mark} border-franc bg-surface`}>
          {ending ? (step.status === 'closed' ? <CloseIcon /> : <CheckIcon />) : <ClockIcon />}
        </span>
        <span className="text-[18px] font-bold sm:text-[17px]">{label}</span>
        <span className="text-sm font-bold sm:font-normal">
          {date && (ending ? t('track.on', { date }) : t('track.since', { date }))}
        </span>
      </li>
    )
  }
  if (step.state === 'done') {
    return (
      <li className={`${row} min-h-10 sm:border-franc sm:bg-ink sm:text-white`}>
        <span aria-hidden="true" className={`${mark} border-franc bg-ink`}>
          <CheckIcon color="#7FD8B6" size={16} />
        </span>
        <span className="font-bold sm:text-[17px]">{label}</span>
        <span className="text-sm sm:text-[#D6D6D6]">{date}</span>
      </li>
    )
  }
  return (
    <li className={`${row} min-h-10 sm:border-franc-dashed sm:bg-surface`}>
      <span aria-hidden="true" className={`${mark} border-franc-dashed`}>
        <span className="hidden sm:inline">
          <BoxIcon />
        </span>
      </span>
      <span className="font-bold text-ink-2 sm:text-[17px] sm:text-ink">{label}</span>
      <span className="text-sm text-ink-2 sm:text-ink">
        {step.status === 'available' && milestone ? t('track.inVersion', { milestone }) : ''}
      </span>
    </li>
  )
}
