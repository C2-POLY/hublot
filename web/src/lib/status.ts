// SPDX-License-Identifier: AGPL-3.0-or-later
import type { Status } from '../api/client'

// The five steps of the parcel-style timeline, in order.
export const TIMELINE: Status[] = ['received', 'planned', 'in_progress', 'in_test', 'available']

export function timelineIndex(status: Status): number {
  return TIMELINE.indexOf(status)
}

export function isActive(status: Status): boolean {
  return status !== 'available' && status !== 'answered' && status !== 'closed'
}

// Pill colours follow the mockups: yellow while being tested, black once available.
export function statusPillClass(status: Status): string {
  switch (status) {
    case 'in_test':
    case 'in_progress':
      return 'bg-accent'
    case 'available':
      return 'bg-ink text-white'
    case 'answered':
      return 'bg-mint'
    default:
      return 'bg-surface'
  }
}
