// SPDX-License-Identifier: AGPL-3.0-or-later
// The triage inbox (docs/design/maquettes/C-Triage.dc.html): every request nobody
// classified yet, across the projects the person may triage. Dense on purpose.
import { useEffect, useId, useMemo, useRef, useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api, ApiError, type Kind, type PortalRequest, type RequestProject, type TriageAction } from '../api/client'
import { useTriageInbox } from '../api/hooks'
import { EmptyState, ErrorState, Loading } from '../components/States'
import { PrivateBadge } from '../components/StatusPill'
import { formatDay } from '../lib/format'

const KINDS: Kind[] = ['problem', 'idea', 'question']
// After these, the request leaves the inbox.
const LEAVES: TriageAction['action'][] = ['accept', 'close']

type Vars = { r: PortalRequest; act: TriageAction }

export function TriagePage() {
  const { t } = useTranslation()
  const inbox = useTriageInbox()
  const queryClient = useQueryClient()
  const [filter, setFilter] = useState<string | null>(null)
  const [selected, setSelected] = useState<string | null>(null)
  const [message, setMessage] = useState('')
  const [failure, setFailure] = useState('')
  const toggles = useRef(new Map<string, HTMLButtonElement>())
  const headingRef = useRef<HTMLHeadingElement>(null)
  // Where focus goes once the list is refreshed: the next request, or the heading.
  const pendingFocus = useRef<string | null>(null)

  const requests = useMemo(() => inbox.data?.requests ?? [], [inbox.data])
  const projects = inbox.data?.projects ?? []
  const counts = useMemo(() => {
    const m = new Map<string, number>()
    for (const r of requests) m.set(r.project.slug, (m.get(r.project.slug) ?? 0) + 1)
    return m
  }, [requests])
  const shown = filter ? requests.filter((r) => r.project.slug === filter) : requests

  useEffect(() => {
    const target = pendingFocus.current
    if (!target) return
    pendingFocus.current = null
    const el = target === 'heading' ? headingRef.current : toggles.current.get(target)
    ;(el ?? headingRef.current)?.focus()
  }, [inbox.data])

  const act = useMutation({
    mutationFn: ({ r, act }: Vars) => api.triage(r.id, act),
    onMutate: () => {
      setMessage('')
      setFailure('')
    },
    onSuccess: async (updated, { r, act }) => {
      const title = r.title
      switch (act.action) {
        case 'kind':
          setMessage(t('triage.done.kind', { title, kind: t(`triage.kinds.${act.kind}`) }))
          break
        case 'transfer': {
          const to = projects.find((p) => p.slug === act.project)
          setMessage(t('triage.done.transfer', { title, project: to?.name ?? act.project }))
          if (filter && filter !== act.project) setFilter(null)
          break
        }
        case 'private':
          setMessage(t(act.private ? 'triage.done.privateOn' : 'triage.done.privateOff', { title }))
          break
        default:
          setMessage(t(`triage.done.${act.action}`, { title }))
      }
      if (LEAVES.includes(act.action) || !updated) {
        const i = shown.findIndex((x) => x.id === r.id)
        const next = shown[i + 1] ?? shown[i - 1]
        pendingFocus.current = next?.id ?? 'heading'
        setSelected(null)
      }
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['triage'] }),
        queryClient.invalidateQueries({ queryKey: ['request', r.id] }),
        queryClient.invalidateQueries({ queryKey: ['requests'] }),
      ])
    },
    onError: (err) => setFailure(t(err instanceof ApiError && err.status === 403 ? 'triage.forbidden' : 'triage.failed')),
  })

  return (
    <div className="flex flex-col gap-3.5">
      <div className="flex flex-wrap items-center justify-between gap-3.5">
        <h1 ref={headingRef} tabIndex={-1} className="m-0 text-[40px] leading-none tracking-[-0.02em] outline-none">
          {t('triage.title')}
        </h1>
        {requests.length > 0 && (
          <div role="group" aria-label={t('triage.filter')} className="flex flex-wrap gap-1.5">
            <Chip pressed={filter === null} onClick={() => setFilter(null)}>
              {t('triage.all', { count: requests.length })}
            </Chip>
            {projects
              .filter((p) => counts.has(p.slug))
              .map((p) => (
                <Chip key={p.slug} pressed={filter === p.slug} onClick={() => setFilter(p.slug)} color={p.color}>
                  {t('triage.project', { name: p.name, count: counts.get(p.slug) })}
                </Chip>
              ))}
          </div>
        )}
      </div>

      {/* One line of fixed height, so a message never pushes the cards down. */}
      <p role="status" aria-live="polite" title={message || undefined} className="m-0 h-6 truncate text-[15px] font-bold leading-6">
        {message}
      </p>
      {failure && (
        <p role="alert" className="m-0 rounded-[12px] border-franc bg-notice px-4 py-2.5 text-[15px] font-bold">
          {failure}
        </p>
      )}

      {inbox.isPending && <Loading />}
      {inbox.isError && <ErrorState message={t('common.error')} onRetry={() => void inbox.refetch()} />}
      {inbox.isSuccess &&
        (shown.length === 0 ? (
          <EmptyState>{filter ? t('triage.emptyFilter') : t('triage.empty')}</EmptyState>
        ) : (
          <ul className="m-0 flex list-none flex-col gap-3.5 p-0">
            {shown.map((r) => (
              <li key={r.id}>
                <TriageCard
                  r={r}
                  open={selected === r.id}
                  onToggle={() => setSelected((s) => (s === r.id ? null : r.id))}
                  toggleRef={(el) => {
                    if (el) toggles.current.set(r.id, el)
                    else toggles.current.delete(r.id)
                  }}
                  destinations={projects.filter((p) => p.slug !== r.project.slug)}
                  busy={act.isPending && act.variables?.r.id === r.id}
                  onAction={(a) => act.mutate({ r, act: a })}
                />
              </li>
            ))}
          </ul>
        ))}
    </div>
  )
}

function Chip({
  pressed,
  onClick,
  color,
  children,
}: {
  pressed: boolean
  onClick: () => void
  color?: string
  children: React.ReactNode
}) {
  const on = color ? 'text-ink' : pressed ? 'bg-ink text-white' : 'bg-surface text-ink'
  return (
    <button
      type="button"
      aria-pressed={pressed}
      onClick={onClick}
      style={color ? { background: color } : undefined}
      className={`h-9 rounded-full border-2 border-ink px-3 text-[13px] font-bold ${on} ${
        pressed && color ? 'shadow-hard-sm' : ''
      }`}
    >
      {children}
    </button>
  )
}

function TriageCard({
  r,
  open,
  onToggle,
  toggleRef,
  destinations,
  busy,
  onAction,
}: {
  r: PortalRequest
  open: boolean
  onToggle: () => void
  toggleRef: (el: HTMLButtonElement | null) => void
  destinations: RequestProject[]
  busy: boolean
  onAction: (a: TriageAction) => void
}) {
  const { t, i18n } = useTranslation()
  const panelId = useId()
  const titleId = useId()
  return (
    <article
      aria-labelledby={titleId}
      aria-busy={busy}
      className={`overflow-hidden rounded-2xl bg-surface ${open ? 'border-franc-strong shadow-hard-md' : 'border-franc'}`}
    >
      <div className="grid grid-cols-[10px_minmax(0,1fr)] items-stretch sm:grid-cols-[10px_minmax(0,1fr)_auto]">
        <span aria-hidden="true" className="row-span-2 border-r-[2.5px] border-ink sm:row-span-1" style={{ background: r.project.color }} />
        <div className="flex min-w-0 flex-col gap-1 px-[18px] py-3.5">
          <p className="m-0 text-[13px] text-ink-2">
            <strong className="text-ink">{r.author.name}</strong>
            {' · '}
            {t('triage.meta', { project: r.project.name, date: formatDay(r.createdAt, i18n.language) })}
          </p>
          <p id={titleId} className="m-0 break-words text-lg font-bold">
            {r.title}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-1.5 px-[18px] pb-3.5 sm:py-3.5">
          <span className="pill">{t(`triage.kinds.${r.kind}`)}</span>
          {r.private && <PrivateBadge />}
          <button
            ref={toggleRef}
            type="button"
            aria-expanded={open}
            aria-controls={panelId}
            aria-label={open ? undefined : t('triage.openLabel', { title: r.title })}
            onClick={onToggle}
            className={`h-[38px] rounded-[10px] border-2 border-ink px-3.5 text-sm font-bold text-ink ${open ? 'bg-surface' : 'bg-accent'}`}
          >
            {open ? t('triage.fold') : t('triage.open')}
          </button>
        </div>
      </div>
      <div id={panelId} hidden={!open}>
        {open && <TriagePanel r={r} destinations={destinations} busy={busy} onAction={onAction} />}
      </div>
    </article>
  )
}

function TriagePanel({
  r,
  destinations,
  busy,
  onAction,
}: {
  r: PortalRequest
  destinations: RequestProject[]
  busy: boolean
  onAction: (a: TriageAction) => void
}) {
  const { t } = useTranslation()
  const moveId = useId()
  const privateId = useId()
  const [to, setTo] = useState('')
  const [closing, setClosing] = useState(false)
  const [reason, setReason] = useState<'not_planned' | 'duplicate'>('not_planned')
  const closeRef = useRef<HTMLButtonElement>(null)
  const firstReason = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (closing) firstReason.current?.focus()
  }, [closing])

  const move = (e: FormEvent) => {
    e.preventDefault()
    if (to) onAction({ action: 'transfer', project: to })
  }
  const cancelClose = () => {
    setClosing(false)
    // The button comes back on the next render.
    requestAnimationFrame(() => closeRef.current?.focus())
  }

  return (
    <div
      role="group"
      aria-label={t('triage.actions', { title: r.title })}
      className="flex flex-col gap-4 border-t-[2.5px] border-ink bg-soft px-[18px] py-4 sm:pl-7"
    >
      <div className="flex flex-wrap items-center gap-3">
        <button
          type="button"
          disabled={busy}
          onClick={() => onAction({ action: 'accept' })}
          className="btn-primary h-11 px-5 text-base"
        >
          {busy ? t('triage.working') : t('triage.accept')}
        </button>
        <Link to={`/demandes/${r.id}`} className="btn-light h-11 px-4 text-[15px]">
          {t('triage.see')}
        </Link>
      </div>

      <fieldset className="m-0 flex flex-wrap items-center gap-2 border-0 p-0">
        <legend className="float-left mr-2 text-sm font-bold">{t('triage.kind')}</legend>
        <div className="flex overflow-hidden rounded-[10px] border-2 border-ink">
          {KINDS.map((k, i) => {
            const current = r.kind === k
            return (
              <button
                key={k}
                type="button"
                aria-pressed={current}
                disabled={busy}
                onClick={() => !current && onAction({ action: 'kind', kind: k })}
                className={`h-[38px] px-3.5 text-sm font-bold ${i > 0 ? 'border-l-2 border-ink' : ''} ${
                  current ? 'bg-ink text-white' : 'bg-surface text-ink'
                }`}
              >
                {t(`triage.kinds.${k}`)}
              </button>
            )
          })}
        </div>
      </fieldset>

      {destinations.length > 0 && (
        <form onSubmit={move} className="flex flex-wrap items-center gap-2">
          <label htmlFor={moveId} className="text-sm font-bold">
            {t('triage.move')}
          </label>
          <select
            id={moveId}
            value={to}
            onChange={(e) => setTo(e.target.value)}
            className="h-[38px] rounded-[10px] border-2 border-ink bg-surface px-2 text-sm font-bold text-ink"
          >
            <option value="">{t('triage.moveChoose')}</option>
            {destinations.map((p) => (
              <option key={p.slug} value={p.slug}>
                {p.name}
              </option>
            ))}
          </select>
          <button
            type="submit"
            disabled={busy || !to}
            className="h-[38px] rounded-[10px] border-2 border-ink bg-surface px-3.5 text-sm font-bold text-ink disabled:opacity-55"
          >
            {t('triage.moveSubmit')}
          </button>
        </form>
      )}

      <label className="flex cursor-pointer items-start gap-3">
        <input
          type="checkbox"
          role="switch"
          checked={r.private}
          disabled={busy}
          aria-describedby={privateId}
          onChange={(e) => onAction({ action: 'private', private: e.target.checked })}
          className="m-0 mt-0.5 h-[22px] w-[22px] shrink-0 accent-ink"
        />
        <span>
          <strong className="block text-[15px]">{t('triage.private')}</strong>
          <span id={privateId} className="text-[13px] text-ink-2">
            {t('triage.privateHint')}
          </span>
        </span>
      </label>

      {closing ? (
        <fieldset className="m-0 flex flex-col gap-2 rounded-[12px] border-franc-dashed bg-surface p-3.5">
          <legend className="px-1 text-sm font-bold">{t('triage.closeReason')}</legend>
          <div className="flex flex-wrap gap-4">
            {(['not_planned', 'duplicate'] as const).map((k, i) => (
              <label key={k} className="flex cursor-pointer items-center gap-2 text-[15px]">
                <input
                  ref={i === 0 ? firstReason : undefined}
                  type="radio"
                  name={`reason-${r.id}`}
                  checked={reason === k}
                  onChange={() => setReason(k)}
                  className="m-0 h-[18px] w-[18px] accent-ink"
                />
                {t(`triage.${k}`)}
              </label>
            ))}
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              disabled={busy}
              onClick={() => onAction({ action: 'close', reason })}
              className="btn-dark h-[38px] px-3.5 text-sm"
            >
              {t('triage.closeConfirm')}
            </button>
            <button type="button" onClick={cancelClose} className="btn-light h-[38px] px-3.5 text-sm">
              {t('common.cancel')}
            </button>
          </div>
        </fieldset>
      ) : (
        <div>
          <button
            ref={closeRef}
            type="button"
            disabled={busy}
            onClick={() => setClosing(true)}
            className="btn-light h-[38px] px-3.5 text-sm"
          >
            {t('triage.close')}
          </button>
        </div>
      )}
    </div>
  )
}
