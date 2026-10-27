// SPDX-License-Identifier: AGPL-3.0-or-later
// The porthole ("hublot") is the brand: one symbol, reused everywhere.
// Each software project travels in its own vehicle; the rocket is the company.
import type { CSSProperties, ReactNode } from 'react'
import type { Vehicle } from '../api/client'

const INK = '#111111'

/** Render once at the app root; every emblem references it with <use href="#hub">. */
export function HubSymbolDefs() {
  return (
    <svg aria-hidden="true" width="0" height="0" style={{ position: 'absolute' }}>
      <defs>
        <symbol id="hub" viewBox="-20 -20 40 40">
          <circle r="18" fill="#ECE7DD" stroke="#111111" strokeWidth="2.5" />
          <circle r="11" fill="currentColor" stroke="#111111" strokeWidth="2.5" />
          <path d="M-6-4a7 7 0 016-4" stroke="#FFFFFF" strokeWidth="2" strokeLinecap="round" fill="none" />
          <g fill="#111111">
            <circle cx="0" cy="-14.5" r="1.6" />
            <circle cx="0" cy="14.5" r="1.6" />
            <circle cx="-14.5" cy="0" r="1.6" />
            <circle cx="14.5" cy="0" r="1.6" />
            <circle cx="10.3" cy="-10.3" r="1.6" />
            <circle cx="-10.3" cy="10.3" r="1.6" />
            <circle cx="-10.3" cy="-10.3" r="1.6" />
            <circle cx="10.3" cy="10.3" r="1.6" />
          </g>
        </symbol>
      </defs>
    </svg>
  )
}

function Hub({ x, y, size, color }: { x: number; y: number; size: number; color: string }) {
  return <use href="#hub" x={x} y={y} width={size} height={size} style={{ color } as CSSProperties} />
}

/** The logo: the porthole alone, yellow glass. */
export function Logo({ size = 30 }: { size?: number }) {
  return (
    <svg aria-hidden="true" viewBox="0 0 40 40" width={size} height={size}>
      <Hub x={0} y={0} size={40} color="#FFE066" />
    </svg>
  )
}

function vehicleBody(vehicle: Vehicle, color: string, compact: boolean): ReactNode {
  switch (vehicle) {
    case 'boat':
      return (
        <>
          <g stroke={INK} strokeLinejoin="round">
            {!compact && <rect x="52" y="20" width="14" height="24" fill="#FFE066" />}
            <rect x="30" y="40" width="62" height="36" rx="6" fill="#FFFFFF" />
            <path d="M10 74H110L96 102H24Z" fill="#FFFFFF" />
            {!compact && <path d="M6 108q9-6 18 0t18 0t18 0t18 0t18 0t18 0" fill="none" />}
          </g>
          <Hub x={44} y={40} size={34} color={color} />
        </>
      )
    case 'submarine':
      return (
        <>
          <g stroke={INK} strokeLinejoin="round">
            {!compact && <path d="M60 40V24h12" fill="none" />}
            {!compact && <rect x="46" y="36" width="30" height="20" rx="6" fill="#FFE066" />}
            <rect x="14" y="50" width="96" height="46" rx="23" fill="#FFE066" />
            {!compact && <path d="M14 73L2 62v22z" fill="#FF9A62" />}
          </g>
          <Hub x={44} y={55} size={36} color={color} />
          {!compact && (
            <g fill="#FFFFFF" stroke={INK} strokeWidth="2">
              <circle cx="102" cy="34" r="5" />
              <circle cx="112" cy="20" r="3.5" />
            </g>
          )}
        </>
      )
    case 'train':
      return (
        <>
          <g stroke={INK} strokeLinejoin="round">
            <path d="M12 36H76Q104 36 110 88H12Z" fill="#FFFFFF" />
            {!compact && <rect x="12" y="72" width="98" height="8" fill="#FF9A62" />}
            {!compact && <path d="M4 100H116" fill="none" />}
          </g>
          {!compact && (
            <g fill={INK}>
              <circle cx="30" cy="94" r="7" />
              <circle cx="88" cy="94" r="7" />
            </g>
          )}
          <Hub x={26} y={38} size={34} color={color} />
        </>
      )
    case 'plane':
      return (
        <>
          <g stroke={INK} strokeLinejoin="round">
            <path d="M20 58L10 26h18l18 28z" fill="#FFB3D1" />
            <path d="M10 64c0-10 14-14 34-14h46c14 0 24 6 26 14-2 8-12 12-26 12H44C24 76 10 74 10 64z" fill="#FFFFFF" />
            <path d="M48 70l18 28h14L68 70z" fill="#FFB3D1" />
          </g>
          <Hub x={58} y={47} size={32} color={color} />
        </>
      )
    case 'rocket':
      return (
        <>
          <g stroke={INK} strokeLinejoin="round">
            {!compact && <path d="M50 92C44 104 60 112 60 118C66 110 74 104 70 92Z" fill="#FF9A62" />}
            <path d="M40 70L28 94H44Z" fill="#FF9A62" />
            <path d="M80 70L92 94H76Z" fill="#FF9A62" />
            <path d="M60 6C78 22 84 44 84 62V92H36V62C36 44 42 22 60 6Z" fill="#FFFFFF" />
            <path d="M48 22C52 14 56 10 60 6C64 10 68 14 72 22Z" fill="#B9AEFF" />
          </g>
          <Hub x={40} y={34} size={40} color={color} />
        </>
      )
    case 'car':
      return (
        <>
          <g stroke={INK} strokeLinejoin="round">
            <path d="M10 86V68q0-8 8-10l16-4 16-20h36l16 20q12 2 12 14v18Z" fill="#FFFFFF" />
            {!compact && <path d="M10 76H114" fill="none" />}
          </g>
          <g fill={INK}>
            <circle cx="34" cy="88" r="10" />
            <circle cx="90" cy="88" r="10" />
          </g>
          <Hub x={52} y={34} size={32} color={color} />
        </>
      )
    case 'bus':
      return (
        <>
          <g stroke={INK} strokeLinejoin="round">
            <rect x="8" y="28" width="104" height="60" rx="10" fill="#FFE066" />
            {!compact && <rect x="56" y="38" width="20" height="20" rx="3" fill="#FFFFFF" />}
            {!compact && <rect x="84" y="38" width="20" height="20" rx="3" fill="#FFFFFF" />}
            {!compact && <path d="M8 70H112" fill="none" />}
          </g>
          <g fill={INK}>
            <circle cx="30" cy="92" r="9" />
            <circle cx="90" cy="92" r="9" />
          </g>
          <Hub x={12} y={32} size={36} color={color} />
        </>
      )
  }
}

const VEHICLES: Vehicle[] = ['boat', 'submarine', 'train', 'plane', 'rocket', 'car', 'bus']

/**
 * A project's vehicle with its porthole in the project's colour.
 * Decorative by default; pass `label` when the emblem carries meaning on its own.
 */
export function VehicleEmblem({
  vehicle,
  color,
  size = 40,
  label,
}: {
  vehicle: Vehicle
  color: string
  size?: number
  label?: string
}) {
  const compact = size < 72
  const known = VEHICLES.includes(vehicle)
  return (
    <svg
      viewBox="0 0 120 120"
      width={size}
      height={size}
      role={label ? 'img' : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
      strokeWidth={compact ? 5 : 3.5}
      style={{ flex: 'none' }}
    >
      {known ? vehicleBody(vehicle, color, compact) : <Hub x={10} y={10} size={100} color={color} />}
    </svg>
  )
}
