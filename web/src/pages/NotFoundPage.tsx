// SPDX-License-Identifier: AGPL-3.0-or-later
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'

export function NotFoundPage() {
  const { t } = useTranslation()
  return (
    <section className="flex max-w-2xl flex-col items-start gap-5 py-10">
      <h1 className="m-0 text-[44px] leading-[0.95] tracking-[-0.03em] sm:text-[56px]">{t('notFound.title')}</h1>
      <p className="m-0 text-lg">{t('notFound.text')}</p>
      <Link to="/" className="btn-primary h-14 px-6 text-lg">
        {t('notFound.home')}
      </Link>
    </section>
  )
}
