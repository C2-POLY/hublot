// SPDX-License-Identifier: AGPL-3.0-or-later
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import type { Kind } from '../api/client'
import { useMyRequests } from '../api/hooks'
import { VehicleEmblem } from '../components/Emblem'
import { RequestRow } from '../components/RequestRow'
import { EmptyState, ErrorState, Loading } from '../components/States'
import { firstName } from '../lib/format'
import { useCurrentUser } from '../lib/me'
import { isActive } from '../lib/status'

const QUICK: { kind: Kind; className: string }[] = [
  { kind: 'problem', className: 'bg-ink text-white hover:text-white' },
  { kind: 'idea', className: 'bg-surface text-ink' },
  { kind: 'question', className: 'bg-surface text-ink' },
]

export function HomePage() {
  const { t } = useTranslation()
  const me = useCurrentUser()
  const requests = useMyRequests()
  const list = requests.data?.requests ?? []
  const active = list.filter((r) => isActive(r.status))
  const available = list.filter((r) => r.status === 'available')
  const projects = me.projects.filter((p) => !p.inbox)

  return (
    <div className="flex flex-col gap-[26px]">
      <section className="flex flex-wrap items-stretch gap-4">
        <div className="flex min-w-0 flex-[2_1_420px] flex-col justify-between gap-[18px] rounded-[22px] border-franc-strong bg-accent p-6 shadow-hard-lg">
          <h1 className="m-0 text-[38px] leading-[0.95] tracking-[-0.03em] sm:text-[50px]">
            {t('home.greeting', { name: firstName(me.name, me.email) })}
          </h1>
          <div className="flex flex-wrap gap-2.5">
            {QUICK.map(({ kind, className }) => (
              <Link
                key={kind}
                to={`/demandes/nouvelle?type=${kind}`}
                className={`inline-flex h-[50px] items-center rounded-[14px] border-franc-strong px-[18px] text-base font-bold no-underline ${className}`}
              >
                {t(`kinds.${kind}.action`)}
              </Link>
            ))}
          </div>
        </div>
        {requests.isSuccess && (
          <div className="grid min-w-0 flex-[1_1_260px] grid-rows-2 gap-2.5">
            <Counter to="/demandes" label={t('home.inProgress')} value={active.length} className="bg-surface" />
            <Counter to="/demandes" label={t('home.available')} value={available.length} className="bg-mint" />
          </div>
        )}
      </section>

      <section aria-labelledby="home-projects">
        <h2 id="home-projects" className="m-0 mb-3 text-[26px]">
          {t('home.myProjects')}
        </h2>
        {projects.length === 0 ? (
          <EmptyState>{t('home.noProjects')}</EmptyState>
        ) : (
          <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-4">
            {projects.map((p) => {
              const count = active.filter((r) => r.project.slug === p.slug).length
              const body = (
                <>
                  <span className="flex items-center justify-between gap-3">
                    <span className="font-display text-[22px] font-extrabold leading-tight">{p.name}</span>
                    <VehicleEmblem vehicle={p.vehicle} color={p.color} size={48} />
                  </span>
                  <span className="text-sm leading-snug">
                    {requests.isSuccess && (
                      <strong className="block">
                        {count > 0 ? t('home.projectActive', { count }) : t('home.projectIdle')}
                      </strong>
                    )}
                    {p.hint}
                  </span>
                </>
              )
              const cls = 'flex flex-col gap-3.5 rounded-[18px] border-franc p-4 text-ink no-underline hover:text-ink'
              return p.canCreate ? (
                <Link
                  key={p.slug}
                  to={`/demandes/nouvelle?logiciel=${encodeURIComponent(p.slug)}`}
                  aria-label={`${p.name}, ${t('home.makeRequest')}`}
                  className={cls}
                  style={{ background: p.color }}
                >
                  {body}
                </Link>
              ) : (
                <div key={p.slug} className={cls} style={{ background: p.color }}>
                  {body}
                </div>
              )
            })}
          </div>
        )}
      </section>

      <section aria-labelledby="home-moving" className="flex flex-col gap-2.5">
        <div className="flex items-baseline justify-between">
          <h2 id="home-moving" className="m-0 text-[26px]">
            {t('home.moving')}
          </h2>
          {list.length > 0 && (
            <Link to="/demandes" className="font-bold">
              {t('home.seeAll')}
            </Link>
          )}
        </div>
        {requests.isPending && <Loading />}
        {requests.isError && <ErrorState message={t('common.error')} onRetry={() => void requests.refetch()} />}
        {requests.isSuccess &&
          (list.length === 0 ? (
            <EmptyState>{t('home.empty')}</EmptyState>
          ) : (
            list.slice(0, 5).map((r) => <RequestRow key={r.id} request={r} />)
          ))}
      </section>
    </div>
  )
}

function Counter({ to, label, value, className }: { to: string; label: string; value: number; className: string }) {
  return (
    <Link
      to={to}
      className={`flex items-center justify-between gap-3 rounded-2xl border-franc px-4 py-3 text-ink no-underline hover:text-ink ${className}`}
    >
      <span className="font-bold">{label}</span>
      <span className="font-display text-[34px] font-extrabold leading-none">{value}</span>
    </Link>
  )
}
