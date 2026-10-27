// SPDX-License-Identifier: AGPL-3.0-or-later
import { useTranslation } from 'react-i18next'
import { Logo } from './Emblem'

export function Loading({ full = false, label }: { full?: boolean; label?: string }) {
  const { t } = useTranslation()
  return (
    <div
      role="status"
      className={`flex items-center justify-center gap-3 font-bold ${full ? 'min-h-screen' : 'py-16'}`}
    >
      <span className="animate-spin motion-reduce:animate-none">
        <Logo size={32} />
      </span>
      {label ?? t('common.loading')}
    </div>
  )
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  const { t } = useTranslation()
  return (
    <div role="alert" className="mx-auto flex max-w-xl flex-col items-start gap-4 rounded-[18px] border-franc bg-notice p-5">
      <p className="m-0 text-[17px]">{message}</p>
      {onRetry && (
        <button type="button" onClick={onRetry} className="btn-light h-11 px-4">
          {t('common.retry')}
        </button>
      )}
    </div>
  )
}

export function EmptyState({ children }: { children: React.ReactNode }) {
  return (
    <p className="m-0 rounded-[16px] border-franc-dashed bg-surface p-6 text-center text-[17px] font-bold">{children}</p>
  )
}
