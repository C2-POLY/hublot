// SPDX-License-Identifier: AGPL-3.0-or-later
import { useState } from 'react'
import { Link, useLocation } from 'react-router'
import { Trans, useTranslation } from 'react-i18next'
import { useMutation } from '@tanstack/react-query'
import { api } from '../api/client'
import { CheckIcon } from '../components/Icons'
import { AuthLayout } from './AuthLayout'

export function LinkSentPage() {
  const { t, i18n } = useTranslation()
  const location = useLocation()
  const email = (location.state as { email?: string } | null)?.email
  const [notice, setNotice] = useState<string | null>(null)

  const resend = useMutation({
    mutationFn: () => api.magicLink(email as string, i18n.language),
    onSuccess: () => setNotice(t('linkSent.resent')),
    onError: () => setNotice(t('common.error')),
  })

  return (
    <AuthLayout>
      <div className="flex flex-col items-center gap-3 text-center">
        <div aria-hidden="true" className="relative mb-6 h-[140px] w-[190px]">
          <span className="absolute inset-0 rounded-[18px] border-franc-strong bg-accent shadow-hard-lg" />
          <svg className="absolute inset-0" width="190" height="140" viewBox="0 0 190 140" fill="none" stroke="#111111" strokeWidth="3" strokeLinejoin="round">
            <path d="M14 22l81 58 81-58" />
          </svg>
          <span className="absolute -right-3.5 -top-3.5 flex h-11 w-11 items-center justify-center rounded-full border-franc-strong bg-mint">
            <CheckIcon />
          </span>
        </div>
        <h1 className="m-0 text-[40px] leading-[0.98] tracking-[-0.03em]">{t('linkSent.title')}</h1>
        <p className="m-0 text-[17px] leading-normal">
          {email ? (
            <Trans i18nKey="linkSent.textWith" values={{ email }} components={{ strong: <strong /> }} />
          ) : (
            t('linkSent.textWithout')
          )}
        </p>
        <div className="mt-8 flex w-full flex-col gap-3">
          {email && (
            <button
              type="button"
              disabled={resend.isPending}
              onClick={() => resend.mutate()}
              className="h-12 bg-transparent text-base font-bold text-ink underline"
            >
              {t('linkSent.resend')}
            </button>
          )}
          <p role="status" className="m-0 min-h-6 font-bold">
            {notice}
          </p>
          <Link to="/connexion" className="font-bold">
            {t('linkSent.otherEmail')}
          </Link>
        </div>
      </div>
    </AuthLayout>
  )
}
