// SPDX-License-Identifier: AGPL-3.0-or-later
import { useId } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import { usePreferences } from '../api/hooks'
import { ErrorState, Loading } from '../components/States'

export function PreferencesPage() {
  const { t } = useTranslation()
  const prefs = usePreferences()
  const queryClient = useQueryClient()
  const hintId = useId()
  const save = useMutation({
    mutationFn: api.setPreferences,
    onSuccess: (data) => queryClient.setQueryData(['preferences'], data),
  })

  return (
    <section className="flex max-w-2xl flex-col gap-5">
      <h1 className="m-0 text-[44px] leading-[0.95] tracking-[-0.03em] sm:text-[56px]">{t('preferences.title')}</h1>
      {prefs.isPending ? (
        <Loading />
      ) : prefs.isError ? (
        <ErrorState onRetry={() => void prefs.refetch()} message={t('common.error')} />
      ) : (
        <article className="overflow-hidden rounded-[18px] border-franc-strong bg-surface shadow-hard-md">
          <div className="border-b-[3px] border-ink bg-lavender px-[18px] py-3.5">
            <h2 className="m-0 text-2xl">{t('preferences.notifications')}</h2>
          </div>
          <div className="flex flex-col gap-3 px-[18px] py-4">
            <label className="flex cursor-pointer items-start justify-between gap-3.5">
              <span>
                <strong className="block text-[15px]">{t('preferences.notifyStatus')}</strong>
                <span id={hintId} className="text-[13px] text-ink-2">
                  {t('preferences.notifyStatusHint')}
                </span>
              </span>
              <input
                type="checkbox"
                role="switch"
                aria-describedby={hintId}
                aria-busy={save.isPending}
                checked={save.isPending ? !!save.variables?.notifyStatus : prefs.data.notifyStatus}
                onChange={(e) => save.mutate({ notifyStatus: e.target.checked })}
                className="m-0 h-[22px] w-[22px] shrink-0 accent-ink"
              />
            </label>
            <p role="status" className="m-0 min-h-5 text-sm font-bold">
              {save.isSuccess && t('preferences.saved')}
            </p>
            {save.isError && (
              <p role="alert" className="m-0 text-sm font-bold">
                {t('preferences.saveFailed')}
              </p>
            )}
          </div>
        </article>
      )}
    </section>
  )
}
