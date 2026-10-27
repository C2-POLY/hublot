// SPDX-License-Identifier: AGPL-3.0-or-later
import { useEffect, useId, useRef, useState } from 'react'
import { Link, NavLink, useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { api, type Me } from '../api/client'
import { LANGS, setLang } from '../i18n'
import { clearDraft } from '../lib/draft'
import { initials } from '../lib/format'
import { Logo } from './Emblem'
import { PlusIcon } from './Icons'

const navBase =
  'inline-flex h-[42px] items-center rounded-xl border-[2.5px] px-3.5 text-[15px] font-bold no-underline'
const navClass = ({ isActive }: { isActive: boolean }) =>
  `${navBase} ${isActive ? 'border-ink bg-ink text-white hover:text-white' : 'border-transparent bg-transparent text-ink'}`

export function Header({ me }: { me: Me }) {
  const { t } = useTranslation()
  const isStaff = me.orgRole === 'owner' || me.orgRole === 'admin'
  const ini = initials(me.name, me.email)

  return (
    <header className="w-full border-b-[3px] border-ink bg-surface">
      <div className="mx-auto flex min-h-[93px] max-w-[1240px] flex-wrap items-center gap-4 px-4 py-3.5 sm:px-7">
        <Link
          to="/"
          aria-label={t('common.homeLabel')}
          className="flex items-center gap-[9px] font-display text-[25px] font-extrabold tracking-[-0.02em] text-ink no-underline"
        >
          <Logo size={34} />
          {t('common.appName')}
        </Link>
        <nav aria-label={t('nav.main')} className="order-last flex w-full flex-wrap items-center gap-1 sm:order-none sm:w-auto sm:flex-1">
          <NavLink to="/" end className={navClass}>
            {t('nav.home')}
          </NavLink>
          <NavLink to="/demandes" end className={navClass}>
            {t('nav.requests')}
          </NavLink>
          {/* Staff links (triage, features, access, settings) and their separator arrive with those screens. */}
        </nav>
        <div className="ml-auto flex items-center gap-4 sm:ml-0">
          <Link
            to="/demandes/nouvelle"
            className="inline-flex h-11 items-center gap-2 rounded-xl border-[2.5px] border-ink bg-accent px-4 text-[15px] font-bold text-ink no-underline shadow-hard-sm hover:text-ink"
          >
            <PlusIcon size={16} />
            {t('nav.newRequest')}
          </Link>
          <AccountMenu me={me} initials={ini} staff={isStaff} />
        </div>
      </div>
    </header>
  )
}

function AccountMenu({ me, initials: ini, staff }: { me: Me; initials: string; staff: boolean }) {
  const { t, i18n } = useTranslation()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuId = useId()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setOpen(false)
        buttonRef.current?.focus()
      }
    }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const logout = async () => {
    try {
      await api.logout()
    } finally {
      clearDraft()
      queryClient.clear()
      navigate('/connexion', { replace: true })
    }
  }

  return (
    <div ref={ref} className="relative">
      <button
        ref={buttonRef}
        type="button"
        aria-label={t('nav.account', { initials: ini })}
        aria-expanded={open}
        aria-controls={menuId}
        onClick={() => setOpen((o) => !o)}
        className={`h-11 w-11 rounded-xl border-[2.5px] border-ink text-[15px] font-bold text-ink ${staff ? 'bg-mint' : 'bg-orange'}`}
      >
        {ini}
      </button>
      {open && (
        <div
          id={menuId}
          className="absolute right-0 top-[52px] z-40 flex w-64 flex-col gap-3 rounded-2xl border-franc-strong bg-surface p-4 shadow-hard-md"
        >
          <div>
            {me.name && <strong className="block">{me.name}</strong>}
            <span className="block break-all text-sm text-ink-2">{me.email}</span>
          </div>
          <fieldset className="m-0 border-0 p-0">
            <legend className="mb-1.5 text-sm font-bold">{t('nav.language')}</legend>
            <div className="flex overflow-hidden rounded-xl border-franc">
              {LANGS.map((l, i) => {
                const active = i18n.language === l
                return (
                  <button
                    key={l}
                    type="button"
                    aria-pressed={active}
                    lang={l}
                    onClick={() => setLang(l)}
                    className={`h-10 flex-1 text-sm font-bold ${i > 0 ? 'border-l-[2.5px] border-ink' : ''} ${active ? 'bg-ink text-white' : 'bg-surface text-ink'}`}
                  >
                    {l === 'fr' ? 'Français' : 'English'}
                  </button>
                )
              })}
            </div>
          </fieldset>
          <button type="button" onClick={() => void logout()} className="btn-dark h-11 px-4">
            {t('nav.logout')}
          </button>
        </div>
      )}
    </div>
  )
}
