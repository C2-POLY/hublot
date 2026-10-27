// SPDX-License-Identifier: AGPL-3.0-or-later
// My requests / All requests (docs/design/maquettes/C-Demandes.dc.html). The filters are
// applied by the server and kept in the address, so a filtered view can be shared.
import { useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import type { Kind, RequestFilter, Status } from '../api/client'
import { useRequests } from '../api/hooks'
import { PlusIcon } from '../components/Icons'
import { RequestRow } from '../components/RequestRow'
import { EmptyState, ErrorState, Loading } from '../components/States'
import { useCurrentUser } from '../lib/me'
import { isActive, STATUSES } from '../lib/status'

type Tab = 'active' | 'done'
const TABS: Tab[] = ['active', 'done']
const KINDS: Kind[] = ['problem', 'idea', 'question']

// Address parameters, in the portal's own words: /demandes?vue=toutes&logiciel=shop&type=idea&statut=planned
const P = { view: 'vue', project: 'logiciel', kind: 'type', status: 'statut' } as const

export function RequestsPage() {
  const { t } = useTranslation()
  const me = useCurrentUser()
  const [params, setParams] = useSearchParams()
  const [tab, setTab] = useState<Tab>('active')
  const [query, setQuery] = useState('')
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([])

  // Unknown values in a hand-edited address are ignored rather than sent.
  const slugs = me.projects.map((p) => p.slug)
  const scope: RequestFilter['scope'] = params.get(P.view) === 'toutes' && slugs.length > 0 ? 'all' : 'mine'
  const rawProject = params.get(P.project) ?? ''
  const project = slugs.includes(rawProject) ? rawProject : scope === 'all' ? slugs[0] : undefined
  const rawKind = params.get(P.kind) as Kind | null
  const kind = rawKind && KINDS.includes(rawKind) ? rawKind : undefined
  const rawStatus = params.get(P.status) as Status | null
  const status = rawStatus && STATUSES.includes(rawStatus) ? rawStatus : undefined
  const filtered = !!(kind || status || (scope === 'mine' && project))

  const requests = useRequests({ scope, project, kind, status })

  const set = (key: string, value: string | undefined) => {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    setParams(next, { replace: true })
  }
  const clear = () => {
    const next = new URLSearchParams()
    if (scope === 'all') {
      next.set(P.view, 'toutes')
      if (project) next.set(P.project, project)
    }
    setParams(next, { replace: true })
  }

  const all = useMemo(() => requests.data?.requests ?? [], [requests.data])
  const groups = useMemo(
    () => ({ active: all.filter((r) => isActive(r.status)), done: all.filter((r) => !isActive(r.status)) }),
    [all],
  )
  // A chosen status already says which group: no tabs then.
  const list = status ? all : groups[tab]
  const q = query.trim().toLocaleLowerCase()
  const shown = list.filter(
    (r) => !q || r.title.toLocaleLowerCase().includes(q) || r.project.name.toLocaleLowerCase().includes(q),
  )

  const onTabKey = (e: KeyboardEvent, i: number) => {
    if (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft') return
    e.preventDefault()
    const next = (i + (e.key === 'ArrowRight' ? 1 : -1) + TABS.length) % TABS.length
    setTab(TABS[next] as Tab)
    tabRefs.current[next]?.focus()
  }

  let empty: string
  if (q) empty = t('requests.noMatch', { query })
  else if (filtered) empty = t('requests.noFilterMatch')
  else if (scope === 'all') empty = t('requests.emptyAll')
  else empty = tab === 'active' ? t('requests.emptyActive') : t('requests.emptyDone')

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <h1 className="m-0 text-[44px] leading-[0.95] tracking-[-0.03em] sm:text-[56px]">
          {t(scope === 'all' ? 'requests.titleAll' : 'requests.title')}
        </h1>
        <Link to="/demandes/nouvelle" className="btn-primary px-5 py-3.5 text-lg">
          <PlusIcon />
          {t('requests.newRequest')}
        </Link>
      </div>

      <div role="group" aria-label={t('requests.filters')} className="flex flex-wrap items-end gap-3">
        {slugs.length > 0 && (
          <div role="radiogroup" aria-label={t('requests.view')} className="flex overflow-hidden rounded-[14px] border-franc bg-surface">
            {(['mine', 'all'] as const).map((v, i) => {
              const on = scope === v
              return (
                <button
                  key={v}
                  type="button"
                  role="radio"
                  aria-checked={on}
                  onClick={() => {
                    const next = new URLSearchParams(params)
                    if (v === 'all') next.set(P.view, 'toutes')
                    else next.delete(P.view)
                    setParams(next, { replace: true })
                  }}
                  className={`h-12 px-4 text-[15px] font-bold ${i > 0 ? 'border-l-[2.5px] border-ink' : ''} ${
                    on ? 'bg-ink text-white' : 'bg-surface text-ink'
                  }`}
                >
                  {t(v === 'mine' ? 'requests.viewMine' : 'requests.viewAll')}
                </button>
              )
            })}
          </div>
        )}
        <FilterSelect
          id="filter-project"
          label={t('requests.filterProject')}
          value={project ?? ''}
          onChange={(v) => set(P.project, v)}
          any={scope === 'mine' ? t('requests.anyProject') : undefined}
          options={me.projects.map((p) => ({ value: p.slug, label: p.name }))}
        />
        <FilterSelect
          id="filter-kind"
          label={t('requests.filterKind')}
          value={kind ?? ''}
          onChange={(v) => set(P.kind, v)}
          any={t('requests.anyKind')}
          options={KINDS.map((k) => ({ value: k, label: t(`triage.kinds.${k}`) }))}
        />
        <FilterSelect
          id="filter-status"
          label={t('requests.filterStatus')}
          value={status ?? ''}
          onChange={(v) => set(P.status, v)}
          any={t('requests.anyStatus')}
          options={STATUSES.map((s) => ({ value: s, label: t(`status.${s}`) }))}
        />
        {filtered && (
          <button type="button" onClick={clear} className="btn-light h-12 px-4">
            {t('requests.clear')}
          </button>
        )}
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        {!status ? (
          <div role="tablist" aria-label={t('requests.filter')} className="flex overflow-hidden rounded-[14px] border-franc bg-surface">
            {TABS.map((id, i) => {
              const selected = tab === id
              return (
                <button
                  key={id}
                  ref={(el) => {
                    tabRefs.current[i] = el
                  }}
                  id={`tab-${id}`}
                  role="tab"
                  type="button"
                  aria-selected={selected}
                  aria-controls="requests-panel"
                  tabIndex={selected ? 0 : -1}
                  onClick={() => setTab(id)}
                  onKeyDown={(e) => onTabKey(e, i)}
                  className={`h-11 px-[18px] text-[15px] font-bold ${i > 0 ? 'border-l-[2.5px] border-ink' : ''} ${
                    selected ? 'bg-ink text-white' : 'bg-surface text-ink'
                  }`}
                >
                  {t(id === 'active' ? 'requests.tabActive' : 'requests.tabDone', { count: groups[id].length })}
                </button>
              )
            })}
          </div>
        ) : (
          <span />
        )}
        <label htmlFor="requests-search" className="sr-only-franc">
          {t('requests.search')}
        </label>
        <input
          id="requests-search"
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t('requests.search')}
          className="field h-12 min-w-0 flex-[0_1_320px] rounded-[14px] text-base"
        />
      </div>

      <div
        id="requests-panel"
        {...(!status ? { role: 'tabpanel', 'aria-labelledby': `tab-${tab}` } : {})}
        className="flex flex-col gap-3"
      >
        {requests.isPending && <Loading />}
        {requests.isError && <ErrorState message={t('common.error')} onRetry={() => void requests.refetch()} />}
        {requests.isSuccess &&
          (shown.length > 0 ? (
            shown.map((r) => <RequestRow key={r.id} request={r} showProgress />)
          ) : (
            <EmptyState>{empty}</EmptyState>
          ))}
      </div>
    </div>
  )
}

function FilterSelect({
  id,
  label,
  value,
  onChange,
  any,
  options,
}: {
  id: string
  label: string
  value: string
  onChange: (value: string | undefined) => void
  any?: string
  options: { value: string; label: string }[]
}) {
  return (
    <div className="flex min-w-0 flex-[1_1_170px] flex-col gap-1 sm:flex-[0_1_220px]">
      <label htmlFor={id} className="text-sm font-bold">
        {label}
      </label>
      <select
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value || undefined)}
        className={`field h-12 rounded-[14px] text-base font-bold ${value && any !== undefined ? 'bg-accent' : ''}`}
      >
        {any !== undefined && <option value="">{any}</option>}
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </div>
  )
}
