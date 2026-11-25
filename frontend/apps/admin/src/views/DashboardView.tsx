import type { AdminView } from '../components/AdminShell'
import type { ManagedEvent } from '../lib/api'

export function DashboardView({ events, loading, onNavigate }: { events: ManagedEvent[]; loading: boolean; onNavigate: (view: AdminView) => void }) {
  const active = events.filter((event) => event.status === 'REGISTRATION_OPEN').length
  const drafts = events.filter((event) => event.status === 'DRAFT').length
  const upcoming = [...events].filter((event) => new Date(event.startsAt).getTime() > Date.now()).sort((a, b) => a.startsAt.localeCompare(b.startsAt)).slice(0, 3)

  return (
    <section className="admin-view">
      <header className="view-heading dashboard-heading">
        <div><p className="eyebrow">Operations overview</p><h1>Good evening, <em>Admin.</em></h1><p>One protected surface for event readiness and explainable matching decisions.</p></div>
        <button className="primary-action" type="button" onClick={() => onNavigate('events')}>Create new event <span aria-hidden="true">＋</span></button>
      </header>

      <div className="metric-grid" aria-label="Event summary">
        <article><span className="metric-icon" aria-hidden="true">▣</span><p>Total events</p><strong>{loading ? '—' : events.length}</strong><small>Across all lifecycle states</small></article>
        <article><span className="metric-icon" aria-hidden="true">◉</span><p>Registration open</p><strong>{loading ? '—' : active}</strong><small>Currently discoverable cohorts</small></article>
        <article><span className="metric-icon" aria-hidden="true">✦</span><p>Drafts awaiting review</p><strong>{loading ? '—' : drafts}</strong><small>Not visible to members</small></article>
      </div>

      <div className="dashboard-columns">
        <section className="panel upcoming-panel">
          <div className="panel-heading"><div><p className="eyebrow">Event calendar</p><h2>Upcoming events</h2></div><button className="text-action" type="button" onClick={() => onNavigate('events')}>View all →</button></div>
          {loading && <p className="empty-state">Loading event operations…</p>}
          {!loading && upcoming.length === 0 && <p className="empty-state">No upcoming events. Create the first draft when the schedule is approved.</p>}
          <div className="upcoming-list">
            {upcoming.map((event) => <article key={event.eventId}><time dateTime={event.startsAt}><strong>{new Date(event.startsAt).toLocaleDateString('en-LK', { day: '2-digit' })}</strong><span>{new Date(event.startsAt).toLocaleDateString('en-LK', { month: 'short' })}</span></time><div><h3>{event.name}</h3><p>{event.broadLocation} · {new Date(event.startsAt).toLocaleTimeString('en-LK', { hour: '2-digit', minute: '2-digit' })}</p></div><span className={`status-badge status-${event.status.toLowerCase()}`}>{event.status.replaceAll('_', ' ')}</span></article>)}
          </div>
        </section>

        <aside className="panel forge-preview">
          <p className="eyebrow">Deterministic engine</p><h2>The matching forge</h2><p>Generate a reproducible proposal from approved fixture snapshots, review safe reasons, then lock and publish.</p>
          <div className="forge-visual" aria-hidden="true"><span>A</span><i>◇</i><span>B</span></div>
          <button className="secondary-action" type="button" onClick={() => onNavigate('matchmaking')}>Open matchmaking</button>
        </aside>
      </div>
    </section>
  )
}
