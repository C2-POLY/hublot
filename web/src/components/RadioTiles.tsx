// SPDX-License-Identifier: AGPL-3.0-or-later
import { useRef, type CSSProperties, type KeyboardEvent, type ReactNode } from 'react'

export interface TileOption<V extends string> {
  value: V
  content: ReactNode
  background: string
  dashed?: boolean
}

/**
 * Big selectable cards (C-NouvelleDemande): an ARIA radiogroup with roving focus.
 * Arrow keys move and select, like native radio buttons.
 */
export function RadioTiles<V extends string>({
  label,
  options,
  value,
  onChange,
  selectedBackground,
}: {
  label: string
  options: TileOption<V>[]
  value: V | undefined
  onChange: (v: V) => void
  /** Optional per-option background once selected (the kind tiles turn coloured). */
  selectedBackground?: (v: V) => string
}) {
  const refs = useRef<(HTMLButtonElement | null)[]>([])
  const selectedIndex = options.findIndex((o) => o.value === value)
  const focusable = selectedIndex >= 0 ? selectedIndex : 0

  const onKey = (e: KeyboardEvent, i: number) => {
    const delta = e.key === 'ArrowRight' || e.key === 'ArrowDown' ? 1 : e.key === 'ArrowLeft' || e.key === 'ArrowUp' ? -1 : 0
    if (!delta) return
    e.preventDefault()
    const next = (i + delta + options.length) % options.length
    const opt = options[next]
    if (!opt) return
    onChange(opt.value)
    refs.current[next]?.focus()
  }

  return (
    <div role="radiogroup" aria-label={label} className="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-3">
      {options.map((o, i) => {
        const checked = o.value === value
        const style: CSSProperties = {
          background: checked && selectedBackground ? selectedBackground(o.value) : o.background,
          borderStyle: o.dashed ? 'dashed' : 'solid',
        }
        return (
          <button
            key={o.value}
            ref={(el) => {
              refs.current[i] = el
            }}
            type="button"
            role="radio"
            aria-checked={checked}
            tabIndex={i === focusable ? 0 : -1}
            onClick={() => onChange(o.value)}
            onKeyDown={(e) => onKey(e, i)}
            style={style}
            className={`flex min-h-[110px] flex-col justify-between gap-3.5 rounded-[18px] border-ink p-[18px] text-left text-ink transition-transform sm:min-h-[130px] ${
              checked ? '-translate-x-0.5 -translate-y-0.5 border-[3px] shadow-hard-md' : 'border-[2.5px]'
            }`}
          >
            {o.content}
          </button>
        )
      })}
    </div>
  )
}
