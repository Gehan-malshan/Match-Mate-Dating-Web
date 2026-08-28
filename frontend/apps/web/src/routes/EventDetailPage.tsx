import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from '@tanstack/react-router'
import { MemberNavigation } from '../components/MemberNavigation'
import { getEvent } from '../lib/event-api'
import './EventDetailPage.css'

function eventDate(value: string, timeZone: string, dateStyle: 'full' | 'medium' = 'medium') { return new Intl.DateTimeFormat('en-LK', { dateStyle, timeStyle: 'short', timeZone }).format(new Date(value)) }
function stateLabel(status: string) { return status === 'REGISTRATION_OPEN' ? 'Registration open' : status === 'REGISTRATION_CLOSED' ? 'Registration closed' : 'Event announced' }

export function EventDetailPage() {
  const { eventId } = useParams({ from: '/events/$eventId' })
  const query = useQuery({ queryKey: ['event', eventId], queryFn: () => getEvent(eventId) })
  if (query.isPending) return <main className="event-detail-page"><MemberNavigation active="events" /><p className="event-detail-state" role="status">Opening this encounter…</p></main>
  if (query.isError) return <main className="event-detail-page"><MemberNavigation active="events" /><section className="event-detail-state"><Link to="/events">← All events</Link><h1>Event unavailable</h1><p>This event may not be published or the Event Service may be offline.</p></section></main>
  const event = query.data
  return <main className="event-detail-page">
    <MemberNavigation active="events" />
    <section className={`event-detail-hero event-detail-hero-${event.eventId.slice(-1)}`}>
      <div className="event-detail-hero-wash" aria-hidden="true" />
      <div className="event-detail-hero-content"><Link className="event-detail-back" to="/events">← Back to events</Link><div className="event-detail-badges"><span>{stateLabel(event.status)}</span><span>{event.broadLocation}</span></div><h1>{event.name}</h1><div className="event-detail-meta"><span>◷ {eventDate(event.startsAt, event.timeZone, 'full')}</span><span>⌖ {event.broadLocation}</span><span>◌ {event.timeZone}</span></div></div>
    </section>
    <div className="event-detail-layout">
      <div className="event-detail-main">
        <section className="event-detail-about"><p className="event-detail-kicker">— About the encounter</p><p>{event.description}</p><p>MatchMate events are hosted introductions designed around consent, respectful conversation, and a shared real-world setting. Member identities and private matching choices are never displayed in this catalogue.</p></section>
        <section className="event-detail-principles" aria-label="Event safeguards"><article><span>✦</span><h2>Curated arrival</h2><p>Event details are shared in stages so public discovery never reveals personal attendee information or an exact venue.</p></article><article><span>◈</span><h2>Privacy-first meeting</h2><p>Contact details remain private. Any future introduction follows MatchMate’s mutual-choice and moderation rules.</p></article></section>
        <section className="event-detail-location"><div><p className="event-detail-kicker">Location details</p><h2>{event.broadLocation}</h2><p>The precise venue is deliberately withheld from public discovery. Confirmed attendees receive the appropriate arrival instructions through a future Booking flow.</p></div><div className="event-detail-map" aria-hidden="true"><span>⌖</span><small>Broad area only</small></div></section>
      </div>
      <aside className="event-detail-aside"><section className="event-detail-ticket"><p className="event-detail-kicker">Entry ticket</p><h2>{event.currency} {event.price}<small> / person</small></h2><dl><div><dt>Configured event size</dt><dd>Up to {event.configuredCapacity} guests</dd></div><div><dt>Registration window</dt><dd>{eventDate(event.registrationOpensAt, event.timeZone)} – {eventDate(event.registrationClosesAt, event.timeZone)}</dd></div><div><dt>Current status</dt><dd className="is-open">{stateLabel(event.status)}</dd></div></dl><Link className="button event-detail-action" to="/login">Log in to prepare your profile <b aria-hidden="true">→</b></Link><p className="event-detail-ticket-note">Booking and payment confirmation are not available in this project phase. Capacity shown here is not a seat guarantee.</p></section><section className="event-detail-preparation"><p className="event-detail-kicker">Before the event</p><ul><li>Complete your private MatchMate profile.</li><li>Review the published event schedule and registration window.</li><li>Keep contact details private until mutual consent.</li></ul></section></aside>
    </div>
  </main>
}
