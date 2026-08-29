import type { ReactNode } from 'react'
import type { AdminAccount } from '../lib/api'

export type AdminView = 'dashboard' | 'events' | 'matchmaking'

const navigation: Array<{ id: AdminView; icon: string; label: string }> = [
  { id: 'dashboard', icon: '⌂', label: 'Overview' },
  { id: 'events', icon: '▣', label: 'Event manager' },
  { id: 'matchmaking', icon: '◇', label: 'Matchmaking' },
]

export function AdminShell({ account, active, onNavigate, onSignOut, children }: {
  account: AdminAccount
  active: AdminView
  onNavigate: (view: AdminView) => void
  onSignOut: () => Promise<void>
  children: ReactNode
}) {
  return (
    <div className="admin-shell">
      <aside className="admin-sidebar">
        <div className="admin-brand" aria-label="MatchMate administration">
          <span className="admin-brand-mark" aria-hidden="true">m</span>
          <div><strong>MatchMate</strong><small>Administration</small></div>
        </div>

        <nav aria-label="Administration navigation">
          {navigation.map((item) => (
            <button key={item.id} type="button" className={active === item.id ? 'is-active' : ''} onClick={() => onNavigate(item.id)}>
              <span aria-hidden="true">{item.icon}</span>{item.label}
            </button>
          ))}
        </nav>

        <div className="admin-authority-note">
          <span aria-hidden="true">⌾</span>
          <p><strong>Restricted workspace</strong>Event creation and matching controls are server-authorized for administrators only.</p>
        </div>

        <div className="admin-identity">
          <span aria-hidden="true">AD</span>
          <div><strong>{account.email ?? 'Administrator'}</strong><small>Administrator</small></div>
          <button type="button" onClick={() => void onSignOut()}>Sign out</button>
        </div>
      </aside>

      <div className="admin-stage">
        <header className="admin-topbar">
          <button className="mobile-brand" type="button" onClick={() => onNavigate('dashboard')}><span aria-hidden="true">m</span> MatchMate</button>
          <div><span className="live-dot" aria-hidden="true" /> Admin session</div>
        </header>
        <main className="admin-content">{children}</main>
      </div>
    </div>
  )
}
