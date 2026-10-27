// SPDX-License-Identifier: AGPL-3.0-or-later
// Thin wrapper over fetch that follows docs/api.md.

export type OrgRole = 'owner' | 'admin' | 'member'
export type ProjectRole = 'viewer' | 'reporter' | 'contributor' | 'triager' | 'manager'
export type Vehicle = 'boat' | 'submarine' | 'train' | 'plane' | 'rocket' | 'car' | 'bus'
export type Kind = 'problem' | 'idea' | 'question'
export type Status =
  | 'received'
  | 'planned'
  | 'in_progress'
  | 'in_test'
  | 'available'
  | 'answered'
  | 'closed'

export interface Project {
  slug: string
  name: string
  hint: string
  color: string
  vehicle: Vehicle
  role: ProjectRole
  canCreate: boolean
  inbox: boolean
}

export interface Me {
  id: string
  email: string
  name: string
  lang: string
  orgRole: OrgRole
  projects: Project[]
}

export type FormField =
  | {
      id: string
      type: 'input' | 'textarea'
      label: string
      description?: string
      placeholder?: string
      required: boolean
    }
  | {
      id: string
      type: 'dropdown'
      label: string
      description?: string
      options: string[]
      multiple: boolean
      required: boolean
    }
  | {
      id: string
      type: 'checkboxes'
      label: string
      description?: string
      options: string[]
      required: boolean
    }
  | { id?: string; type: 'markdown'; value: string }

export interface Form {
  kind: Kind
  template: string
  name: string
  description?: string
  fields: FormField[]
}

export interface Attachment {
  id: string
  name: string
  contentType: string
  size: number
  url: string
}

export interface RequestProject {
  slug: string
  name: string
  color: string
  vehicle: Vehicle
}

export interface PortalRequest {
  id: string
  project: RequestProject
  kind: Kind
  title: string
  status: Status
  private: boolean
  number: number
  createdAt: string
  author: { name: string }
  attachments: Attachment[]
}

export type FieldValue = string | string[]

export interface NewRequest {
  project: string
  kind: Kind
  title: string
  fields: Record<string, FieldValue>
  attachments: string[]
  private: boolean
}

export class ApiError extends Error {
  status: number
  code: string
  fields: Record<string, string>

  constructor(status: number, code: string, message: string, fields: Record<string, string> = {}) {
    super(message)
    this.status = status
    this.code = code
    this.fields = fields
  }
}

// Set by the app so any 401 sends the user back to the sign-in page.
let onUnauthenticated: () => void = () => {}
export function setUnauthenticatedHandler(fn: () => void) {
  onUnauthenticated = fn
}

async function request<T>(method: string, path: string, body?: unknown, opts: { silent401?: boolean } = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  let payload: BodyInit | undefined
  if (method !== 'GET') headers['X-Hublot'] = '1'
  if (body instanceof FormData) {
    payload = body
  } else if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }

  const res = await fetch(path, { method, headers, body: payload, credentials: 'same-origin' })

  if (res.status === 204) return undefined as T
  const text = await res.text()
  let data: unknown = undefined
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = undefined
    }
  }

  if (!res.ok) {
    const err = (data as { error?: { code?: string; message?: string; fields?: Record<string, string> } } | undefined)?.error
    if (res.status === 401 && !opts.silent401) onUnauthenticated()
    throw new ApiError(res.status, err?.code ?? 'internal', err?.message ?? res.statusText, err?.fields ?? {})
  }
  return data as T
}

export const api = {
  magicLink: (email: string, lang: string) => request<object>('POST', '/api/auth/magic-link', { email, lang }),
  verify: (token: string) => request<{ user: Me }>('POST', '/api/auth/verify', { token }, { silent401: true }),
  logout: () => request<void>('POST', '/api/auth/logout'),
  me: () => request<Me>('GET', '/api/me'),
  forms: (slug: string) => request<{ forms: Form[] }>('GET', `/api/projects/${encodeURIComponent(slug)}/forms`),
  uploadAttachment: (file: File) => {
    const fd = new FormData()
    fd.append('file', file, file.name)
    return request<Attachment>('POST', '/api/attachments', fd)
  },
  deleteAttachment: (id: string) => request<void>('DELETE', `/api/attachments/${encodeURIComponent(id)}`),
  createRequest: (body: NewRequest) => request<PortalRequest>('POST', '/api/requests', body),
  myRequests: () => request<{ requests: PortalRequest[] }>('GET', '/api/requests?scope=mine'),
  getRequest: (id: string) => request<PortalRequest>('GET', `/api/requests/${encodeURIComponent(id)}`),
}
