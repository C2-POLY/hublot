// SPDX-License-Identifier: AGPL-3.0-or-later
import { useEffect } from 'react'
import { Outlet } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useMe } from '../api/hooks'
import { setLang, storedLang } from '../i18n'
import { Header } from './Header'
import { Footer } from './Footer'
import { ErrorState, Loading } from './States'

/** Signed-in layout: shared header, page, footer. A 401 redirects to /connexion. */
export function AppShell() {
  const { t } = useTranslation()
  const me = useMe()

  // Follow the account's language unless this browser picked one explicitly.
  useEffect(() => {
    if (me.data?.lang && !storedLang()) setLang(me.data.lang, false)
  }, [me.data?.lang])

  if (me.isPending) return <Loading full />
  if (me.isError) {
    return (
      <div className="min-h-screen p-7">
        <ErrorState onRetry={() => void me.refetch()} message={t('common.error')} />
      </div>
    )
  }

  return (
    <div className="flex min-h-screen flex-col">
      <a
        href="#main"
        className="sr-only-franc focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50 focus:h-auto focus:w-auto btn-light px-4 py-2"
      >
        {t('nav.main')}
      </a>
      <Header me={me.data} />
      <main id="main" tabIndex={-1} className="mx-auto w-full max-w-[1240px] flex-1 px-4 pb-14 pt-7 outline-none sm:px-7">
        <Outlet context={me.data} />
      </main>
      <Footer />
    </div>
  )
}
