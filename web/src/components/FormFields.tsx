// SPDX-License-Identifier: AGPL-3.0-or-later
// Renders a repository's GitHub issue form, field by field.
import { useTranslation } from 'react-i18next'
import type { FieldValue, FormField } from '../api/client'
import { Markdown } from './Markdown'

export function fieldDomId(id: string) {
  return `nd-field-${id}`
}

function Label({ htmlFor, label, required }: { htmlFor?: string; label: string; required: boolean }) {
  const { t } = useTranslation()
  const content = (
    <>
      {label}{' '}
      <span className="font-normal text-ink-2">{required ? t('wizard.required') : t('wizard.optional')}</span>
    </>
  )
  return htmlFor ? (
    <label htmlFor={htmlFor} className="text-[15px] font-bold">
      {content}
    </label>
  ) : (
    <legend className="mb-1 text-[15px] font-bold">{content}</legend>
  )
}

function Help({ id, text }: { id: string; text?: string }) {
  if (!text) return null
  return (
    <p id={id} className="m-0 text-sm text-ink-2">
      {text}
    </p>
  )
}

function ErrorText({ id, error }: { id: string; error?: string }) {
  if (!error) return null
  return (
    <p id={id} className="m-0 text-sm font-bold text-[#b3261e]">
      {error}
    </p>
  )
}

export function FormFields({
  fields,
  values,
  errors,
  onChange,
}: {
  fields: FormField[]
  values: Record<string, FieldValue>
  errors: Record<string, string>
  onChange: (id: string, v: FieldValue) => void
}) {
  return (
    <>
      {fields.map((f, i) => {
        if (f.type === 'markdown') {
          return <Markdown key={`md-${i}`} source={f.value} />
        }
        const dom = fieldDomId(f.id)
        const helpId = `${dom}-help`
        const errId = `${dom}-error`
        const error = errors[f.id]
        const describedBy = [f.description ? helpId : '', error ? errId : ''].filter(Boolean).join(' ') || undefined

        if (f.type === 'input' || f.type === 'textarea') {
          const v = typeof values[f.id] === 'string' ? (values[f.id] as string) : ''
          const common = {
            id: dom,
            value: v,
            placeholder: f.placeholder,
            required: f.required,
            'aria-required': f.required,
            'aria-invalid': error ? true : undefined,
            'aria-describedby': describedBy,
          }
          return (
            <div key={f.id} className="flex flex-col gap-2">
              <Label htmlFor={dom} label={f.label} required={f.required} />
              <Help id={helpId} text={f.description} />
              {f.type === 'input' ? (
                <input {...common} onChange={(e) => onChange(f.id, e.target.value)} className="field h-[52px]" />
              ) : (
                <textarea
                  {...common}
                  onChange={(e) => onChange(f.id, e.target.value)}
                  className="field h-[150px] resize-y py-3 text-base"
                />
              )}
              <ErrorText id={errId} error={error} />
            </div>
          )
        }

        if (f.type === 'dropdown') {
          const raw = values[f.id]
          if (f.multiple) {
            const selected = Array.isArray(raw) ? raw : []
            return (
              <div key={f.id} className="flex flex-col gap-2">
                <Label htmlFor={dom} label={f.label} required={f.required} />
                <Help id={helpId} text={f.description} />
                <select
                  id={dom}
                  multiple
                  value={selected}
                  aria-required={f.required}
                  aria-invalid={error ? true : undefined}
                  aria-describedby={describedBy}
                  onChange={(e) => onChange(f.id, Array.from(e.target.selectedOptions, (o) => o.value))}
                  className="field py-2"
                >
                  {f.options.map((o) => (
                    <option key={o} value={o}>
                      {o}
                    </option>
                  ))}
                </select>
                <ErrorText id={errId} error={error} />
              </div>
            )
          }
          return (
            <div key={f.id} className="flex flex-col gap-2">
              <Label htmlFor={dom} label={f.label} required={f.required} />
              <Help id={helpId} text={f.description} />
              <SelectOne id={dom} field={f} value={typeof raw === 'string' ? raw : ''} onChange={(v) => onChange(f.id, v)} describedBy={describedBy} invalid={!!error} />
              <ErrorText id={errId} error={error} />
            </div>
          )
        }

        if (f.type !== 'checkboxes') return null
        const selected = Array.isArray(values[f.id]) ? (values[f.id] as string[]) : []
        return (
          <fieldset key={f.id} id={dom} tabIndex={-1} aria-describedby={describedBy} className="m-0 flex flex-col gap-2 border-0 p-0">
            <Label label={f.label} required={f.required} />
            <Help id={helpId} text={f.description} />
            {f.options.map((o, oi) => {
              const cid = `${dom}-${oi}`
              const checked = selected.includes(o)
              return (
                <label key={o} htmlFor={cid} className="flex items-start gap-3 text-base">
                  <input
                    id={cid}
                    type="checkbox"
                    checked={checked}
                    onChange={() => onChange(f.id, checked ? selected.filter((s) => s !== o) : [...selected, o])}
                    className="mt-0.5 h-5 w-5 flex-none accent-[#111111]"
                  />
                  {o}
                </label>
              )
            })}
            <ErrorText id={errId} error={error} />
          </fieldset>
        )
      })}
    </>
  )
}

function SelectOne({
  id,
  field,
  value,
  onChange,
  describedBy,
  invalid,
}: {
  id: string
  field: Extract<FormField, { type: 'dropdown' }>
  value: string
  onChange: (v: string) => void
  describedBy?: string
  invalid: boolean
}) {
  const { t } = useTranslation()
  return (
    <select
      id={id}
      value={value}
      aria-required={field.required}
      aria-invalid={invalid ? true : undefined}
      aria-describedby={describedBy}
      onChange={(e) => onChange(e.target.value)}
      className="field h-[52px]"
    >
      <option value="">{t('wizard.selectPlaceholder')}</option>
      {field.options.map((o) => (
        <option key={o} value={o}>
          {o}
        </option>
      ))}
    </select>
  )
}

/** Client-side check of required fields, mirroring what the server enforces. */
export function missingRequired(fields: FormField[], values: Record<string, FieldValue>): string[] {
  const missing: string[] = []
  for (const f of fields) {
    if (f.type === 'markdown' || !f.required) continue
    const v = values[f.id]
    const empty = Array.isArray(v) ? v.length === 0 : !v || !v.trim()
    if (empty) missing.push(f.id)
  }
  return missing
}
