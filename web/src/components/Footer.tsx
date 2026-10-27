// SPDX-License-Identifier: AGPL-3.0-or-later
import { useTranslation } from 'react-i18next'

export const SOURCE_URL = 'https://github.com/C2-POLY/hublot'

/** AGPL-3.0 §13: the interface offers its own source code. */
export function Footer() {
  const { t } = useTranslation()
  return (
    <footer className="border-t-[3px] border-ink bg-surface">
      <div className="mx-auto flex max-w-[1240px] flex-wrap items-center justify-between gap-3 px-4 py-5 text-sm sm:px-7">
        <span>{t('footer.license')}</span>
        <a href={SOURCE_URL} className="font-bold" rel="noopener">
          {t('footer.source')}
        </a>
      </div>
    </footer>
  )
}
