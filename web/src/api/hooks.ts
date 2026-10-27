// SPDX-License-Identifier: AGPL-3.0-or-later
import { useQuery } from '@tanstack/react-query'
import { api, ApiError, type RequestFilter } from './client'

const noRetryOnClientError = (count: number, err: unknown) =>
  !(err instanceof ApiError && err.status >= 400 && err.status < 500) && count < 2

export function useMe() {
  return useQuery({ queryKey: ['me'], queryFn: api.me, retry: noRetryOnClientError, staleTime: 60_000 })
}

export function usePreferences() {
  return useQuery({ queryKey: ['preferences'], queryFn: api.preferences, retry: noRetryOnClientError })
}

export function useMyRequests() {
  return useQuery({ queryKey: ['requests', 'mine'], queryFn: api.myRequests, retry: noRetryOnClientError })
}

export function useRequests(filter: RequestFilter, enabled = true) {
  return useQuery({
    queryKey: ['requests', filter],
    queryFn: () => api.requests(filter),
    enabled,
    retry: noRetryOnClientError,
  })
}

export function useHome() {
  return useQuery({ queryKey: ['home'], queryFn: api.home, retry: noRetryOnClientError })
}

export function useRequest(id: string) {
  return useQuery({ queryKey: ['request', id], queryFn: () => api.getRequest(id), retry: noRetryOnClientError })
}

export function useForms(slug: string | undefined) {
  return useQuery({
    queryKey: ['forms', slug],
    queryFn: () => api.forms(slug as string),
    enabled: !!slug,
    retry: noRetryOnClientError,
    staleTime: 5 * 60_000,
  })
}

export function useTriageInbox() {
  return useQuery({ queryKey: ['triage'], queryFn: api.triageInbox, retry: noRetryOnClientError })
}
