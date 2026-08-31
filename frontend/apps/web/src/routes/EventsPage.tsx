import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { MemberNavigation } from '../components/MemberNavigation'
import { listEvents } from '../lib/event-api'
import './EventsPage.css'

function formatDate(value: string, timeZone: string) { return new Intl.DateTimeFormat('en-LK', { dateStyle: 'medium', timeStyle: 'short', timeZone }).format(new Date(value)) }
function label(status: string) { return status === 'REGISTRATION_OPEN' ? 'Registration open' : status === 'REGISTRATION_CLOSED' ? 'Registration closed' : 'Announced' }

export function EventsPage() {
  const query = useQuery({ queryKey: ['events'], queryFn: () => listEvents() })
  return <main className="events-page event-experience-page">
    <MemberNavigation active="events" />
    <section className="events-hero event-experience-hero"><p className="eyebrow"><span /> Curated event catalogue</p><h1>Available <em>encounters.</em></h1><p>Explore confirmed MatchMate gatherings. Exact venues, member identities, and booking availability stay protected until the right stage.</p><div className="event-trust-pills"><span>Upcoming only</span><span>Privacy-first check-in</span><span>Colombo time</span></div></section>
    <section className="event-catalog event-experience-catalog" aria-busy={query.isPending}>
      {query.isPending && <p className="catalog-message" role="status">Loading confirmed events…</p>}
      {query.isError && <div className="catalog-message" role="alert"><strong>Events are temporarily unavailable.</strong><span>Start the local Event Service, then try again.</span><button className="button button-ghost" onClick={() => void query.refetch()}>Try again</button></div>}
      {query.data?.items.length === 0 && <div className="catalog-message"><strong>No confirmed events yet.</strong><span>New dates will appear here after an organizer publishes them.</span></div>}
      <div className="event-experience-grid">{query.data?.items.map(event => <article className="event-experience-card" key={event.eventId}>
        <div className="event-experience-wash" aria-hidden="true" /><div className="event-card-content"><p className="event-card-status"><span aria-hidden="true" />{label(event.status)}</p><p className="event-card-location">{event.broadLocation}</p><h2>{event.name}</h2><p className="event-card-date">{formatDate(event.startsAt, event.timeZone)}</p><p className="event-card-description">{event.description}</p><div className="event-card-footer"><span>{event.currency} {event.price}</span><Link to="/events/$eventId" params={{ eventId: event.eventId }}>View details <b aria-hidden="true">→</b></Link></div></div>
      </article>)}</div>
    </section>
  </main>
}
