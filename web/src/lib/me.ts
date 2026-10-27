// SPDX-License-Identifier: AGPL-3.0-or-later
import { useOutletContext } from 'react-router'
import type { Me } from '../api/client'

/** The signed-in user, provided by AppShell to every page inside it. */
export function useCurrentUser(): Me {
  return useOutletContext<Me>()
}
