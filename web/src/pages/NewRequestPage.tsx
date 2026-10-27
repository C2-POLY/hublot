// SPDX-License-Identifier: AGPL-3.0-or-later
// Guided request, one question at a time (C-NouvelleDemande, C-Depot 1–4 on mobile):
// software → kind → "tell us" (title, the repository's form fields, screenshots) → sent.
import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { DRAFT_KEY, clearDraft } from '../lib/draft'
import { Link, useSearchParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api, ApiError, type Attachment, type FieldValue, type Kind, type PortalRequest, type Project } from '../api/client'
import { useForms } from '../api/hooks'
import { Captures } from '../components/Captures'
import { VehicleEmblem } from '../components/Emblem'
import { FormFields, fieldDomId, missingRequired } from '../components/FormFields'
import { CheckIcon } from '../components/Icons'
import { RadioTiles } from '../components/RadioTiles'
import { EmptyState, ErrorState, Loading } from '../components/States'
import { useCurrentUser } from '../lib/me'

type Step = 1 | 2 | 3

interface Draft {
  step: Step
  project?: string
  kind?: Kind
  title: string
  fields: Record<string, FieldValue>
  attachments: Attachment[]
  private: boolean
}

const KINDS: Kind[] = ['problem', 'idea', 'question']
const KIND_STYLE: Record<Kind, { color: string; mark: string }> = {
  problem: { color: '#FFE066', mark: '!' },
  idea: { color: '#7FD8B6', mark: '+' },
  question: { color: '#FF9A62', mark: '?' },
}

const emptyDraft = (): Draft => ({ step: 1, title: '', fields: {}, attachments: [], private: false })

function loadDraft(): Draft | null {
  try {
    const raw = localStorage.getItem(DRAFT_KEY)
    if (!raw) return null
    const d = JSON.parse(raw) as Partial<Draft>
    return { ...emptyDraft(), ...d, step: d.step === 2 || d.step === 3 ? d.step : 1 }
  } catch {
    return null
  }
}

function saveDraft(d: Draft) {
  try {
    localStorage.setItem(DRAFT_KEY, JSON.stringify(d))
  } catch {
    // No storage (private mode, quota): the flow still works, only the draft is lost on close.
  }
}


function isKind(v: string | null): v is Kind {
  return v !== null && (KINDS as string[]).includes(v)
}

/** Query parameters (?type=, ?logiciel=) start a fresh request; otherwise the saved draft resumes. */
function initialDraft(params: URLSearchParams, projects: Project[]): Draft {
  const type = params.get('type')
  const slug = params.get('logiciel')
  if (type || slug) {
    const project = projects.find((p) => p.slug === slug)?.slug
    return { ...emptyDraft(), project, kind: isKind(type) ? type : undefined, step: project ? 2 : 1 }
  }
  const d = loadDraft()
  if (!d) return emptyDraft()
  if (d.project && !projects.some((p) => p.slug === d.project)) return { ...d, project: undefined, kind: undefined, fields: {}, step: 1 }
  return d
}

export function NewRequestPage() {
  const { t } = useTranslation()
  const me = useCurrentUser()
  const [params, setParams] = useSearchParams()
  const queryClient = useQueryClient()

  const projects = useMemo(() => {
    const open = me.projects.filter((p) => p.canCreate)
    return [...open.filter((p) => !p.inbox), ...open.filter((p) => p.inbox)]
  }, [me.projects])

  const [draft, setDraft] = useState<Draft>(() => initialDraft(params, projects))
  const [sent, setSent] = useState<PortalRequest | null>(null)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)

  // Keep the URL clean once its parameters are applied, so a reload resumes the draft.
  useEffect(() => {
    if (params.has('type') || params.has('logiciel')) setParams({}, { replace: true })
  }, [params, setParams])

  useEffect(() => {
    if (!sent) saveDraft(draft)
  }, [draft, sent])

  const forms = useForms(draft.project)
  const availableKinds = useMemo(
    () => KINDS.filter((k) => forms.data?.forms.some((f) => f.kind === k)),
    [forms.data],
  )
  const form = forms.data?.forms.find((f) => f.kind === draft.kind)
  const project = projects.find((p) => p.slug === draft.project)

  const update = (patch: Partial<Draft>) => setDraft((d) => ({ ...d, ...patch }))
  const projectName = (p: Project | undefined) => (p ? (p.inbox ? t('wizard.inboxName') : p.name) : '')

  const submit = useMutation({
    mutationFn: () =>
      api.createRequest({
        project: draft.project as string,
        kind: draft.kind as Kind,
        title: draft.title.trim(),
        fields: draft.fields,
        attachments: draft.attachments.map((a) => a.id),
        private: draft.private,
      }),
    onSuccess: (r) => {
      clearDraft()
      setSent(r)
      void queryClient.invalidateQueries({ queryKey: ['requests'] })
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 422) {
        const mapped: Record<string, string> = {}
        for (const [k, v] of Object.entries(err.fields)) mapped[k] = v === 'required' ? t('wizard.fieldRequired') : t('wizard.fieldInvalid')
        setErrors(mapped)
        setFormError(t('wizard.fixErrors'))
        focusFirstError(Object.keys(mapped))
      } else {
        setFormError(t('common.error'))
      }
    },
  })

  const focusFirstError = (ids: string[]) => {
    const first = ids[0]
    if (!first) return
    const el = document.getElementById(first === 'title' ? 'nd-title' : fieldDomId(first))
    el?.focus()
  }

  const next = (e?: FormEvent) => {
    e?.preventDefault()
    if (draft.step === 1 && draft.project) update({ step: 2 })
    else if (draft.step === 2 && draft.kind && availableKinds.includes(draft.kind)) update({ step: 3 })
    else if (draft.step === 3 && form) {
      const missing = missingRequired(form.fields, draft.fields)
      if (!draft.title.trim()) missing.unshift('title')
      if (missing.length) {
        setErrors(Object.fromEntries(missing.map((id) => [id, t('wizard.fieldRequired')])))
        setFormError(t('wizard.fixErrors'))
        focusFirstError(missing)
        return
      }
      setErrors({})
      setFormError(null)
      submit.mutate()
    }
  }

  const back = () => {
    setErrors({})
    setFormError(null)
    update({ step: (draft.step - 1 || 1) as Step })
  }

  const restart = () => {
    setSent(null)
    setDraft(emptyDraft())
    setErrors({})
    setFormError(null)
  }

  const shownStep = sent ? 4 : draft.step
  const stepLabels = [t('wizard.stepProject'), t('wizard.stepKind'), t('wizard.stepDetails'), t('wizard.stepSent')]
  const stepValue = (n: number) => {
    const done = shownStep > n
    const cur = shownStep === n
    if (n === 1) return projectName(project) || t('wizard.toChoose')
    if (n === 2) return done && draft.kind ? t(`kinds.${draft.kind}.name`) : cur ? t('wizard.current') : ''
    if (n === 3) return done ? t('wizard.done') : cur ? t('wizard.current') : ''
    return ''
  }

  const canNext =
    (draft.step === 1 && !!draft.project) ||
    (draft.step === 2 && !!draft.kind && availableKinds.includes(draft.kind)) ||
    (draft.step === 3 && !!form && !submit.isPending)

  return (
    <div className="flex flex-wrap items-start gap-6">
      <aside aria-label={t('wizard.steps')} className="flex min-w-0 flex-[1_1_240px] flex-col gap-2.5 md:max-w-[300px]">
        <h1 className="m-0 mb-2 text-[30px] leading-none tracking-[-0.02em] sm:text-[34px]">{t('wizard.title')}</h1>
        <ol className="m-0 hidden list-none flex-col gap-2 p-0 md:flex">
          {stepLabels.map((label, i) => {
            const n = i + 1
            const done = shownStep > n
            const cur = shownStep === n
            return (
              <li
                key={label}
                aria-current={cur ? 'step' : undefined}
                className={`flex items-center gap-3 rounded-[14px] px-3.5 py-3 ${
                  cur
                    ? 'border-franc-strong bg-accent shadow-hard'
                    : done
                      ? 'border-franc bg-surface'
                      : 'border-[2.5px] border-transparent'
                }`}
              >
                <span
                  className={`flex h-[34px] w-[34px] flex-none items-center justify-center rounded-[10px] border-franc font-extrabold ${
                    done ? 'bg-ink' : 'bg-surface'
                  }`}
                >
                  {done ? <CheckIcon size={18} color="#7FD8B6" /> : n}
                </span>
                <span>
                  <strong className="block">{label}</strong>
                  <span className="text-[13px]">{stepValue(n)}</span>
                </span>
              </li>
            )
          })}
        </ol>
        {!sent && (
          <div className="md:hidden">
            <p className="m-0 mb-2 text-sm font-bold">{t('wizard.progress', { step: draft.step })}</p>
            <div aria-hidden="true" className="flex gap-1.5">
              {[1, 2, 3].map((n) => (
                <span
                  key={n}
                  className={`h-3 flex-1 rounded-md border-2 border-ink ${n < draft.step ? 'bg-ink' : n === draft.step ? 'bg-accent' : 'bg-surface'}`}
                />
              ))}
            </div>
          </div>
        )}
        {!sent && <p className="m-0 mt-2 text-[13px] text-ink-2">{t('wizard.draftSaved')}</p>}
      </aside>

      <form
        onSubmit={next}
        noValidate
        className="flex min-h-[520px] min-w-0 flex-[999_1_640px] flex-col gap-5 rounded-[22px] border-franc-strong bg-surface p-5 shadow-hard-lg sm:p-7"
      >
        {sent ? (
          <Sent request={sent} kind={draft.kind} projectLabel={projectName(project) || sent.project.name} onRestart={restart} />
        ) : (
          <>
            {draft.step === 1 && (
              <>
                <h2 className="m-0 text-[32px] leading-none tracking-[-0.02em] sm:text-[40px]">{t('wizard.projectQuestion')}</h2>
                {projects.some((p) => p.inbox) && <p className="m-0 text-base">{t('wizard.projectHelp')}</p>}
                {projects.length === 0 ? (
                  <EmptyState>{t('wizard.noProject')}</EmptyState>
                ) : (
                  <RadioTiles
                    label={t('wizard.projectGroup')}
                    value={draft.project}
                    onChange={(slug) => {
                      if (slug !== draft.project) update({ project: slug, fields: {} })
                    }}
                    options={projects.map((p) => ({
                      value: p.slug,
                      background: p.inbox ? '#FFFFFF' : p.color,
                      dashed: p.inbox,
                      content: (
                        <>
                          <span className="flex items-start justify-between gap-2">
                            <span className="font-display text-[22px] font-extrabold leading-tight">{projectName(p)}</span>
                            {!p.inbox && <VehicleEmblem vehicle={p.vehicle} color={p.color} size={44} />}
                          </span>
                          <span className="text-sm">{p.inbox ? t('wizard.inboxHint') : p.hint}</span>
                        </>
                      ),
                    }))}
                  />
                )}
              </>
            )}

            {draft.step === 2 && (
              <>
                {project && <p className="m-0 text-[15px] font-bold">{projectName(project)}</p>}
                <h2 className="m-0 text-[32px] leading-none tracking-[-0.02em] sm:text-[40px]">{t('wizard.kindQuestion')}</h2>
                {forms.isPending && <Loading />}
                {forms.isError && <ErrorState message={t('common.error')} onRetry={() => void forms.refetch()} />}
                {forms.isSuccess &&
                  (availableKinds.length === 0 ? (
                    <EmptyState>{t('wizard.noForms')}</EmptyState>
                  ) : (
                    <RadioTiles
                      label={t('wizard.kindGroup')}
                      value={draft.kind && availableKinds.includes(draft.kind) ? draft.kind : undefined}
                      onChange={(k) => {
                        if (k !== draft.kind) update({ kind: k, fields: {} })
                      }}
                      selectedBackground={(k) => KIND_STYLE[k].color}
                      options={availableKinds.map((k) => ({
                        value: k,
                        background: '#FFFFFF',
                        content: (
                          <>
                            <span
                              aria-hidden="true"
                              className="flex h-11 w-11 items-center justify-center rounded-xl border-franc font-display text-2xl font-extrabold"
                              style={{ background: KIND_STYLE[k].color }}
                            >
                              {KIND_STYLE[k].mark}
                            </span>
                            <span className="font-display text-2xl font-extrabold">{t(`kinds.${k}.name`)}</span>
                            <span className="text-[15px]">{t(`kinds.${k}.hint`)}</span>
                          </>
                        ),
                      }))}
                    />
                  ))}
              </>
            )}

            {draft.step === 3 && (
              <>
                <h2 className="m-0 text-[32px] leading-none tracking-[-0.02em] sm:text-[40px]">{t('wizard.tellUs')}</h2>
                {forms.isPending && <Loading />}
                {forms.isSuccess && !form && <EmptyState>{t('wizard.noForms')}</EmptyState>}
                {form && (
                  <div className="flex flex-wrap gap-5">
                    <div className="flex min-w-0 flex-[3_1_360px] flex-col gap-4">
                      <div className="flex flex-col gap-2">
                        <label htmlFor="nd-title" className="text-[15px] font-bold">
                          {t('wizard.titleLabel')}
                        </label>
                        <input
                          id="nd-title"
                          value={draft.title}
                          maxLength={200}
                          onChange={(e) => update({ title: e.target.value })}
                          placeholder={t('wizard.titlePlaceholder')}
                          aria-required
                          aria-invalid={errors.title ? true : undefined}
                          aria-describedby={errors.title ? 'nd-title-error' : undefined}
                          className="field h-[52px]"
                        />
                        {errors.title && (
                          <p id="nd-title-error" className="m-0 text-sm font-bold text-[#b3261e]">
                            {errors.title}
                          </p>
                        )}
                      </div>
                      <FormFields
                        fields={form.fields}
                        values={draft.fields}
                        errors={errors}
                        onChange={(id, v) => setDraft((d) => ({ ...d, fields: { ...d.fields, [id]: v } }))}
                      />
                      <label className="flex items-start gap-3 text-[15px]">
                        <input
                          type="checkbox"
                          checked={draft.private}
                          onChange={(e) => update({ private: e.target.checked })}
                          className="mt-0.5 h-5 w-5 flex-none accent-[#111111]"
                        />
                        {t('wizard.privateLabel')}
                      </label>
                    </div>
                    <div className="min-w-0 flex-[2_1_260px]">
                      <Captures
                        attachments={draft.attachments}
                        onAdd={(a) => setDraft((d) => ({ ...d, attachments: [...d.attachments, a] }))}
                        onRemove={(id) => setDraft((d) => ({ ...d, attachments: d.attachments.filter((x) => x.id !== id) }))}
                      />
                    </div>
                  </div>
                )}
              </>
            )}

            {formError && (
              <p role="alert" className="m-0 rounded-xl border-franc bg-notice px-4 py-3 font-bold">
                {formError}
              </p>
            )}

            <div className="mt-auto flex items-center justify-between gap-3 border-t-[2.5px] border-dashed border-ink pt-[18px]">
              {draft.step > 1 ? (
                <button type="button" onClick={back} className="btn-light h-[52px] px-[18px] text-base">
                  {t('common.back')}
                </button>
              ) : (
                <Link to="/" className="font-bold">
                  {t('common.cancel')}
                </Link>
              )}
              <button type="submit" disabled={!canNext} className="btn-primary h-[58px] px-[26px] text-xl">
                {draft.step === 3 ? (submit.isPending ? t('wizard.sending') : t('wizard.send')) : t('common.next')}
              </button>
            </div>
          </>
        )}
      </form>
    </div>
  )
}

function Sent({
  request,
  kind,
  projectLabel,
  onRestart,
}: {
  request: PortalRequest
  kind: Kind | undefined
  projectLabel: string
  onRestart: () => void
}) {
  const { t } = useTranslation()
  return (
    <>
      <div className="flex flex-wrap items-center gap-7">
        <span
          aria-hidden="true"
          className="flex h-[120px] w-[120px] flex-none -rotate-6 items-center justify-center rounded-[32px] border-franc-strong bg-mint shadow-hard-lg"
        >
          <svg width="58" height="58" viewBox="0 0 46 46" fill="none" stroke="#111111" strokeWidth="5" strokeLinecap="round" strokeLinejoin="round">
            <path d="M10 24l9 9 17-20" />
          </svg>
        </span>
        <div className="min-w-0 flex-[1_1_320px]">
          <h2 role="status" className="m-0 mb-2.5 text-[42px] leading-[0.95] tracking-[-0.03em] sm:text-[52px]">
            {t('wizard.doneTitle')}
          </h2>
          <p className="m-0 text-[19px] leading-normal">{t('wizard.doneText')}</p>
        </div>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-3.5 rounded-2xl border-franc bg-soft p-[18px]">
        <span className="min-w-0">
          <span className="block text-[13px] font-bold">
            {projectLabel} · {t(`kinds.${request.kind ?? kind}.name`)}
          </span>
          <strong className="break-words text-[19px]">{request.title}</strong>
        </span>
        <span className="pill bg-accent px-3 py-1.5 text-sm">{t(`status.${request.status}`)}</span>
      </div>
      <div className="flex flex-wrap gap-3">
        <Link to={`/demandes/${request.id}`} className="btn-dark h-14 rounded-2xl px-[22px] font-display text-[19px] font-extrabold">
          {t('wizard.follow')}
        </Link>
        <button type="button" onClick={onRestart} className="btn-light h-14 rounded-2xl px-5 text-base">
          {t('wizard.another')}
        </button>
      </div>
    </>
  )
}
