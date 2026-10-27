// SPDX-License-Identifier: AGPL-3.0-or-later
import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useMutation } from '@tanstack/react-query'
import { api, ApiError } from '../api/client'
import { AuthLayout } from './AuthLayout'

export function LoginPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [error, setError] = useState<string | null>(null)

  const send = useMutation({
    mutationFn: (address: string) => api.magicLink(address, i18n.language),
    onSuccess: (_d, address) => navigate('/connexion/lien-envoye', { state: { email: address } }),
    onError: (err) => {
      if (err instanceof ApiError && err.status === 422) setError(t('login.invalidEmail'))
      else if (err instanceof ApiError && err.status === 429) setError(t('login.rateLimited'))
      else setError(t('common.error'))
    },
  })

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const address = email.trim()
    if (!address || !address.includes('@')) {
      setError(t('login.invalidEmail'))
      return
    }
    setError(null)
    send.mutate(address)
  }

  return (
    <AuthLayout>
      <form onSubmit={submit} noValidate className="flex flex-col gap-[18px]">
        <h1 className="m-0 text-[40px] leading-[0.95] tracking-[-0.03em] sm:text-[48px]">{t('login.title')}</h1>
        <p className="m-0 text-lg leading-normal">{t('login.intro')}</p>
        <label htmlFor="login-email" className="mt-1.5 font-bold">
          {t('login.email')}
        </label>
        <input
          id="login-email"
          type="email"
          name="email"
          autoComplete="email"
          inputMode="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? 'login-error' : 'login-note'}
          className="field h-[58px] rounded-[14px] border-[3px] text-lg"
        />
        {error && (
          <p id="login-error" role="alert" className="m-0 font-bold">
            {error}
          </p>
        )}
        <button type="submit" disabled={send.isPending} className="btn-primary h-[60px] text-xl">
          {send.isPending ? t('login.sending') : t('login.submit')}
        </button>
        <p id="login-note" className="m-0 mt-1 text-sm text-ink-2">
          {t('login.note')}
        </p>
      </form>
    </AuthLayout>
  )
}
