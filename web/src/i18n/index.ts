// SPDX-License-Identifier: AGPL-3.0-or-later
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import fr from './fr'
import en from './en'

export const LANGS = ['fr', 'en'] as const
export type Lang = (typeof LANGS)[number]

const STORAGE_KEY = 'hublot:lang'

function normalize(lang: string | null | undefined): Lang | undefined {
  const base = lang?.toLowerCase().split('-')[0]
  return LANGS.find((l) => l === base)
}

export function storedLang(): Lang | undefined {
  try {
    return normalize(localStorage.getItem(STORAGE_KEY))
  } catch {
    return undefined
  }
}

function initialLang(): Lang {
  return storedLang() ?? normalize(navigator.language) ?? 'fr'
}

/** Switch language. `remember` keeps the choice for this browser. */
export function setLang(lang: string, remember = true) {
  const l = normalize(lang)
  if (!l) return
  if (remember) {
    try {
      localStorage.setItem(STORAGE_KEY, l)
    } catch {
      // Storage may be unavailable (private mode); the switch still applies.
    }
  }
  void i18n.changeLanguage(l)
}

i18n.on('languageChanged', (lng) => {
  document.documentElement.lang = lng
})

void i18n.use(initReactI18next).init({
  resources: { fr: { translation: fr }, en: { translation: en } },
  lng: initialLang(),
  fallbackLng: 'fr',
  interpolation: { escapeValue: false },
  returnObjects: false,
})

document.documentElement.lang = i18n.language

export default i18n
