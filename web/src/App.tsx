// SPDX-License-Identifier: AGPL-3.0-or-later
import { useEffect } from 'react'
import { Route, Routes, useLocation, useNavigate } from 'react-router'
import { useQueryClient } from '@tanstack/react-query'
import { setUnauthenticatedHandler } from './api/client'
import { HubSymbolDefs } from './components/Emblem'
import { AppShell } from './components/AppShell'
import { LoginPage } from './pages/LoginPage'
import { LinkSentPage } from './pages/LinkSentPage'
import { VerifyPage } from './pages/VerifyPage'
import { HomePage } from './pages/HomePage'
import { NewRequestPage } from './pages/NewRequestPage'
import { RequestsPage } from './pages/RequestsPage'
import { RequestPage } from './pages/RequestPage'
import { NotFoundPage } from './pages/NotFoundPage'

export default function App() {
  const navigate = useNavigate()
  const location = useLocation()
  const queryClient = useQueryClient()

  useEffect(() => {
    setUnauthenticatedHandler(() => {
      queryClient.removeQueries({ queryKey: ['me'] })
      if (!window.location.pathname.startsWith('/connexion')) navigate('/connexion', { replace: true })
    })
  }, [navigate, queryClient])

  // Move focus to the page on navigation so screen readers announce the new view.
  useEffect(() => {
    document.getElementById('main')?.focus({ preventScroll: true })
    window.scrollTo(0, 0)
  }, [location.pathname])

  return (
    <>
      <HubSymbolDefs />
      <Routes>
        <Route path="/connexion" element={<LoginPage />} />
        <Route path="/connexion/lien-envoye" element={<LinkSentPage />} />
        <Route path="/connexion/verifier" element={<VerifyPage />} />
        <Route element={<AppShell />}>
          <Route index element={<HomePage />} />
          <Route path="demandes" element={<RequestsPage />} />
          <Route path="demandes/nouvelle" element={<NewRequestPage />} />
          <Route path="demandes/:id" element={<RequestPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </>
  )
}
