// SPDX-License-Identifier: AGPL-3.0-or-later
import { useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useMyRequests } from '../api/hooks'
import { PlusIcon } from '../components/Icons'
import { RequestRow } from '../components/RequestRow'
import { EmptyState, ErrorState, Loading } from '../components/States'
import { isActive } from '../lib/status'

type Tab = 'active' | 'done'
const TABS: Tab[] = ['active', 'done']

export function RequestsPage() {
  const { t } = useTranslation()
  const requests = useMyRequests()
  const [tab, setTab] = useState<Tab>('active')
  const [query, setQuery] = useState('')
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([])

  const all = useMemo(() => requests.data?.requests ?? [], [requests.data])
  const groups = useMemo(
    () => ({ active: all.filter((r) => isActive(r.status)), done: all.filter((r) => !isActive(r.status)) }),
    [all],
  )
  const q = query.trim().toLocaleLowerCase()
  const shown = groups[tab].filter(
    (r) => !q || r.title.toLocaleLowerCase().includes(q) || r.project.name.toLocaleLowerCase().includes(q),
  )

  const onTabKey = (e: KeyboardEvent, i: number) => {
    if (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft') return
    e.preventDefault()
    const next = (i + (e.key === 'ArrowRight' ? 1 : -1) + TABS.length) % TABS.length
    setTab(TABS[next] as Tab)
    tabRefs.current[next]?.focus()
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <h1 className="m-0 text-[44px] leading-[0.95] tracking-[-0.03em] sm:text-[56px]">{t('requests.title')}</h1>
        <Link to="/demandes/nouvelle" className="btn-primary px-5 py-3.5 text-lg">
          <PlusIcon />
          {t('requests.newRequest')}
        </Link>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
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

      <div id="requests-panel" role="tabpanel" aria-labelledby={`tab-${tab}`} className="flex flex-col gap-3">
        {requests.isPending && <Loading />}
        {requests.isError && <ErrorState message={t('common.error')} onRetry={() => void requests.refetch()} />}
        {requests.isSuccess &&
          (shown.length > 0 ? (
            shown.map((r) => <RequestRow key={r.id} request={r} showProgress />)
          ) : (
            <EmptyState>
              {q ? t('requests.noMatch', { query }) : tab === 'active' ? t('requests.emptyActive') : t('requests.emptyDone')}
            </EmptyState>
          ))}
      </div>
    </div>
  )
}
