// SPDX-License-Identifier: AGPL-3.0-or-later
// Small line icons drawn in the Franc stroke style. Always decorative.

type P = { size?: number; color?: string }

export function PlusIcon({ size = 20, color = '#111111' }: P) {
  return (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="3.2" strokeLinecap="round">
      <path d="M12 4v16M4 12h16" />
    </svg>
  )
}

export function CheckIcon({ size = 20, color = '#111111', width = 3 }: P & { width?: number }) {
  return (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 20 20" fill="none" stroke={color} strokeWidth={width} strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 10.5l4 4 8-9" />
    </svg>
  )
}

export function ClockIcon({ size = 20 }: P) {
  return (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 20 20" fill="none" stroke="#111111" strokeWidth="2.6" strokeLinecap="round">
      <circle cx="10" cy="10" r="7" />
      <path d="M10 6v4l2.5 2" />
    </svg>
  )
}

export function BoxIcon({ size = 20 }: P) {
  return (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 20 20" fill="none" stroke="#111111" strokeWidth="2.6" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 8l7-4 7 4v7l-7 4-7-4z" />
    </svg>
  )
}

export function CloseIcon({ size = 16 }: P) {
  return (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="#111111" strokeWidth="3.2" strokeLinecap="round">
      <path d="M6 6l12 12M18 6L6 18" />
    </svg>
  )
}

export function FileIcon({ size = 28 }: P) {
  return (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="#111111" strokeWidth="2.4" strokeLinejoin="round">
      <path d="M6 3h8l5 5v13H6z" />
      <path d="M14 3v5h5" />
    </svg>
  )
}
