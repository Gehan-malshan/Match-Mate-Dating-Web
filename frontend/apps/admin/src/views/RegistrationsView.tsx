import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { listAllRegistrations, listArchivedEvents, listEventRegistrations, type ManagedEvent } from '../lib/api'

const pageSize = 25

export function RegistrationsView({ events, eventsLoading, eventsError, selectedEventId, onSelectedEventIdChange }: {
  events: ManagedEvent[]
  eventsLoading: boolean
  eventsError: boolean
  selectedEventId: string
  onSelectedEventIdChange: (eventId: string) => void
}) {
  const [offset, setOffset] = useState(0)
  const registrations = useQuery({
    queryKey: ['admin-registrations', selectedEventId, offset],
    queryFn: () => selectedEventId ? listEventRegistrations(selectedEventId, offset) : listAllRegistrations(offset),
  })
  const deletedEvents = useQuery({ queryKey: ['admin-archived-events'], queryFn: listArchivedEvents })
  const allEvents = [...events, ...(deletedEvents.data?.items ?? [])]
  const eventNames = new Map(allEvents.map((event) => [event.eventId, event.name]))
  const items = registrations.data?.items ?? []

  return <section className="admin-view">
    <header className="view-heading">
      <div><p className="eyebrow">Protected booking records</p><h1>Event <em>registrations.</em></h1><p>Review members who reserved seats, online payment holds, and bookings needing review across your events.</p></div>
    </header>
    <section className="panel registrations-panel" aria-label="Event registrations">
      <div className="panel-heading"><div><p className="eyebrow">Registration ledger</p><h2>{selectedEventId ? eventNames.get(selectedEventId) ?? 'Selected event' : 'All events'}</h2><p>Confirmed means a seat is reserved. For pay-at-venue bookings, payment has not been collected in the app.</p></div></div>
      <div className="registration-controls">
        <label htmlFor="registration-event-filter">Event
          <select id="registration-event-filter" value={selectedEventId} onChange={(change) => { onSelectedEventIdChange(change.target.value); setOffset(0) }}>
            <option value="">All events</option>
            {allEvents.map((event) => <option key={event.eventId} value={event.eventId}>{event.name}{event.archivedAt ? ' (deleted)' : ''}</option>)}
          </select>
        </label>
        <button className="ghost-action" type="button" disabled={registrations.isFetching} onClick={() => void registrations.refetch()}>Refresh</button>
      </div>
      {eventsLoading && <p className="registration-hint" role="status">Loading event names…</p>}
      {eventsError && <p className="registration-hint is-error" role="alert">Event names are unavailable; booking IDs remain visible.</p>}
      {registrations.isPending && <p className="empty-state" role="status">Loading registrations…</p>}
      {registrations.isError && <p className="empty-state is-error" role="alert">Could not load registrations. Check your administrator session and the Booking and Account services.</p>}
      {!registrations.isPending && !registrations.isError && items.length === 0 && <p className="empty-state">{offset ? 'No more registrations on this page.' : 'No bookings have been created for this selection yet.'}</p>}
      {items.length > 0 && <div className="registrations-scroll"><table>
        <thead><tr><th>Member</th><th>Event</th><th>Booked</th><th>Status</th><th>Payment</th><th>Amount</th><th>Booking reference</th></tr></thead>
        <tbody>{items.map((item) => <tr key={item.bookingId}>
          <td><strong>{item.nickname || 'Member unavailable'}</strong><small>{item.email || item.accountId}</small></td>
          <td><strong>{eventNames.get(item.eventId) ?? 'Event unavailable'}</strong><small>{item.eventId}</small></td>
          <td>{new Date(item.createdAt).toLocaleString()}</td>
          <td><span className={`status-badge status-${item.state.toLowerCase()}`}>{item.state.replaceAll('_', ' ')}</span>{item.confirmedAt && <small>Confirmed {new Date(item.confirmedAt).toLocaleString()}</small>}</td>
          <td>{item.paymentMethod === 'AT_VENUE' ? 'Pay at venue' : 'Online'}{item.paymentMethod === 'AT_VENUE' && item.state === 'CONFIRMED' && <small>Not collected in app</small>}</td>
          <td>{item.currency} {item.amount}</td>
          <td><code>{item.bookingId}</code></td>
        </tr>)}</tbody>
      </table></div>}
      <div className="registrations-pagination">
        <button className="ghost-action" type="button" disabled={offset === 0 || registrations.isFetching} onClick={() => setOffset(Math.max(0, offset - pageSize))}>Previous</button>
        <span>Page {Math.floor(offset / pageSize) + 1}</span>
        <button className="ghost-action" type="button" disabled={!registrations.data?.hasMore || registrations.isFetching} onClick={() => setOffset(offset + pageSize)}>Next</button>
      </div>
    </section>
  </section>
}
