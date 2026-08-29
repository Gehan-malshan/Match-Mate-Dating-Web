import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AdminShell, type AdminView } from './components/AdminShell'
import { AdminAccount, listManagedEvents, logout, restoreAdminSession } from './lib/api'
import { DashboardView } from './views/DashboardView'
import { EventsView } from './views/EventsView'
import { MatchmakingView } from './views/MatchmakingView'

const memberLoginUrl = import.meta.env.VITE_MEMBER_LOGIN_URL ?? 'http://localhost:5173/login'

export function App() {
  const [account, setAccount] = useState<AdminAccount>()
  const [checkingSession, setCheckingSession] = useState(true)
  const [activeView, setActiveView] = useState<AdminView>('dashboard')

  useEffect(() => {
    restoreAdminSession()
      .then(setAccount)
      .finally(() => setCheckingSession(false))
  }, [])

  useEffect(() => {
    if (!checkingSession && !account) {
      window.location.replace(memberLoginUrl)
    }
  }, [account, checkingSession])

  if (checkingSession) {
    return <main className="admin-loading"><span aria-hidden="true" /><p>Opening the protected workspace…</p></main>
  }

  if (!account) {
    return <main className="admin-loading"><span aria-hidden="true" /><p>Redirecting to MatchMate sign in…</p></main>
  }

  return <AdminWorkspace account={account} activeView={activeView} onViewChange={setActiveView} onSignedOut={() => setAccount(undefined)} />
}

function AdminWorkspace({ account, activeView, onViewChange, onSignedOut }: {
  account: AdminAccount
  activeView: AdminView
  onViewChange: (view: AdminView) => void
  onSignedOut: () => void
}) {
  const eventsQuery = useQuery({ queryKey: ['admin-events'], queryFn: listManagedEvents })
  const events = eventsQuery.data?.items ?? []

  return (
    <AdminShell
      account={account}
      active={activeView}
      onNavigate={onViewChange}
      onSignOut={async () => { await logout(); onSignedOut() }}
    >
      {activeView === 'dashboard' && <DashboardView events={events} loading={eventsQuery.isPending} onNavigate={onViewChange} />}
      {activeView === 'events' && <EventsView account={account} events={events} loading={eventsQuery.isPending} error={eventsQuery.isError} />}
      {activeView === 'matchmaking' && <MatchmakingView events={events} loadingEvents={eventsQuery.isPending} />}
    </AdminShell>
  )
}
