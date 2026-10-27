// SPDX-License-Identifier: AGPL-3.0-or-later

// The unsent request draft lives in this browser only. It is cleared on sign-out so the
// next person on a shared computer never resumes someone else's request.
export const DRAFT_KEY = 'hublot:draft:v1'

export function clearDraft() {
  try {
    localStorage.removeItem(DRAFT_KEY)
  } catch {
    // Nothing to clear.
  }
}
