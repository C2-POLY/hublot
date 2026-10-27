// SPDX-License-Identifier: AGPL-3.0-or-later
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Logo } from '../components/Emblem'
import { Footer } from '../components/Footer'
import { VoyageScene } from '../components/VoyageScene'

/** Signed-out layout (C-Connexion): black brand panel on the left, the form on the right. */
export function AuthLayout({ children }: { children: ReactNode }) {
  const { t } = useTranslation()
  return (
    <div className="flex min-h-screen flex-col">
      <div className="flex flex-1 flex-wrap">
        <section
          aria-hidden="true"
          className="flex min-h-[260px] min-w-0 flex-[1_1_420px] flex-col justify-between gap-8 bg-ink p-8 sm:min-h-[360px] sm:p-12"
        >
          <span className="flex items-center gap-2.5 font-display text-[28px] font-extrabold text-white">
            <Logo size={36} />
            {t('common.appName')}
          </span>
          <VoyageScene className="max-w-[640px] rounded-[24px] border-[3px] border-white" />
          <p className="m-0 max-w-[460px] font-display text-[30px] font-extrabold leading-none text-white sm:text-[38px]">
            {t('login.tagline')}
          </p>
        </section>
        <main id="main" tabIndex={-1} className="flex min-w-0 flex-[1_1_480px] items-center justify-center px-4 py-12 outline-none sm:px-8">
          <div className="w-full max-w-[440px]">{children}</div>
        </main>
      </div>
      <Footer />
    </div>
  )
}
