// SPDX-License-Identifier: AGPL-3.0-or-later

export function formatDay(iso: string, lang: string, long = false): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return new Intl.DateTimeFormat(lang, { day: 'numeric', month: long ? 'long' : 'short' }).format(d)
}

export function formatSize(bytes: number, lang: string): string {
  const units = lang.startsWith('fr') ? ['o', 'Ko', 'Mo'] : ['B', 'KB', 'MB']
  let v = bytes
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${new Intl.NumberFormat(lang, { maximumFractionDigits: i === 0 ? 0 : 1 }).format(v)} ${units[i]}`
}

export function initials(name: string, email: string): string {
  const source = name.trim() || email.split('@')[0] || '?'
  const parts = source.split(/[\s._-]+/).filter(Boolean)
  const letters = parts.length > 1 ? (parts[0]?.[0] ?? '') + (parts[1]?.[0] ?? '') : source.slice(0, 2)
  return letters.toUpperCase()
}

export function firstName(name: string, email: string): string {
  const n = name.trim()
  if (n) return n.split(/\s+/)[0] ?? n
  const local = email.split('@')[0] ?? ''
  const first = local.split(/[._-]/)[0] ?? local
  return first.charAt(0).toUpperCase() + first.slice(1)
}
