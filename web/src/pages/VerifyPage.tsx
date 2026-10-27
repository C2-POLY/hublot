// SPDX-License-Identifier: AGPL-3.0-or-later
import { useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import { setLang, storedLang } from '../i18n'
import { Loading } from '../components/States'
import { takeReturnTo } from '../lib/returnTo'
import { AuthLayout } from './AuthLayout'

// A token is single use: never send it twice, even when React mounts twice in dev.
const attempted = new Set<string>()

export function VerifyPage() {
  const { t } = useTranslation()
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const token = params.get('token') ?? ''
  const [failed, setFailed] = useState(!token)

  useEffect(() => {
    if (!token || attempted.has(token)) return
    attempted.add(token)
    api
      .verify(token)
      .then(({ user }) => {
        queryClient.setQueryData(['me'], user)
        if (!storedLang()) setLang(user.lang, false)
        const next = takeReturnTo()
        // Files are served by the server, not by the SPA router.
        if (next?.startsWith('/files/')) window.location.replace(next)
        else navigate(next ?? '/', { replace: true })
      })
      .catch(() => setFailed(true))
  }, [token, navigate, queryClient])

  return (
    <AuthLayout>
      {failed ? (
        <div className="flex flex-col gap-[18px]">
          <h1 className="m-0 text-[40px] leading-[0.95] tracking-[-0.03em]">{t('verify.failedTitle')}</h1>
          <p className="m-0 text-lg leading-normal">{t('verify.failedText')}</p>
          <Link to="/connexion" className="btn-primary h-[60px] text-xl">
            {t('verify.again')}
          </Link>
        </div>
      ) : (
        <Loading label={t('verify.checking')} />
      )}
    </AuthLayout>
  )
}
