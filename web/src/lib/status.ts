// SPDX-License-Identifier: AGPL-3.0-or-later
import type { PortalRequest, Status } from '../api/client'

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

export const STATUSES: Status[] = [...TIMELINE, 'answered', 'closed']

export type StepState = 'done' | 'current' | 'todo'

export interface Step {
  status: Status
  state: StepState
  // When the request reached this step (absent for steps not reached yet).
  at?: string
}

/**
 * The parcel timeline of a request. A request on its way follows the five steps
 * (a question only two: received, answered). An answered or closed request ends
 * there: the steps it went through, then that ending, never "available".
 */
export function parcelSteps(r: Pick<PortalRequest, 'kind' | 'status' | 'createdAt' | 'history'>): Step[] {
  const history = r.history ?? []
  const reachedAt = (s: Status): string | undefined => {
    for (let i = history.length - 1; i >= 0; i--) if (history[i]?.status === s) return history[i]?.at
    return s === 'received' ? r.createdAt : undefined
  }
  const middle = (s: Status) => s !== 'received' && TIMELINE.includes(s) && s !== 'available'
  const wentThroughMiddle = history.some((h) => middle(h.status)) || middle(r.status)

  if (r.status === 'answered' || r.status === 'closed') {
    const before = TIMELINE.filter((s) => s === 'received' || (s !== 'available' && history.some((h) => h.status === s)))
    return [
      ...before.map((s): Step => ({ status: s, state: 'done', at: reachedAt(s) })),
      { status: r.status, state: 'done', at: reachedAt(r.status) },
    ]
  }
  const path: Status[] = r.kind === 'question' && !wentThroughMiddle ? ['received', 'answered'] : TIMELINE
  const current = Math.max(0, path.indexOf(r.status))
  return path.map((s, i): Step => {
    if (i < current || (i === current && s === 'available')) return { status: s, state: 'done', at: reachedAt(s) }
    if (i === current) return { status: s, state: 'current', at: reachedAt(s) }
    return { status: s, state: 'todo' }
  })
}

/** The i18n key of the plain sentence under the timeline. */
export function statusSentenceKey(r: Pick<PortalRequest, 'status' | 'milestone'>): string {
  const withVersion = r.milestone && ['planned', 'in_progress', 'in_test', 'available'].includes(r.status)
  return withVersion ? `statusLineVersion.${r.status}` : `statusLine.${r.status}`
}
