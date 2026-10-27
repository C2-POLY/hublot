// SPDX-License-Identifier: AGPL-3.0-or-later
import { useTranslation } from 'react-i18next'
import type { Status } from '../api/client'
import { statusPillClass } from '../lib/status'

export function StatusPill({ status, large = false }: { status: Status; large?: boolean }) {
  const { t } = useTranslation()
  return <span className={`pill ${statusPillClass(status)} ${large ? 'px-3 py-1.5 text-sm' : ''}`}>{t(`status.${status}`)}</span>
}

export function PrivateBadge() {
  const { t } = useTranslation()
  return <span className="pill bg-pink px-2 py-0.5 text-xs">{t('common.private')}</span>
}
