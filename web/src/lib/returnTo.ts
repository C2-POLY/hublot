// SPDX-License-Identifier: AGPL-3.0-or-later

// Where to go once signed in, e.g. the attachment a developer opened from a GitHub issue.
// Kept in localStorage because the sign-in link usually opens in another tab.
const KEY = 'hublot:return-to'

/** Only paths of this site: never another origin, never a protocol-relative URL. */
function safe(path: string | null): string | null {
  return path && path.startsWith('/') && !path.startsWith('//') && !path.startsWith('/\\') ? path : null
}

export function rememberReturnTo(path: string | null) {
  try {
    const p = safe(path)
    if (p) localStorage.setItem(KEY, p)
    else localStorage.removeItem(KEY)
  } catch {
    // Without storage the person lands on the home page.
  }
}

export function takeReturnTo(): string | null {
  try {
    const p = safe(localStorage.getItem(KEY))
    localStorage.removeItem(KEY)
    return p
  } catch {
    return null
  }
}
