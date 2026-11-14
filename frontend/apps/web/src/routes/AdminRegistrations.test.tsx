import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { RegistrationsView } from '../../../admin/src/views/RegistrationsView'
import { listAllRegistrations, listEventRegistrations, type ManagedEvent } from '../../../admin/src/lib/api'

vi.mock('../../../admin/src/lib/api', () => ({
  listArchivedEvents: vi.fn().mockResolvedValue({ items: [] }),
  listAllRegistrations: vi.fn().mockResolvedValue({ items: [{ bookingId: 'booking-a', accountId: 'member-a', nickname: 'Asha', email: 'asha@example.test', eventId: 'event-a', state: 'CONFIRMED', paymentMethod: 'AT_VENUE', amount: '2500.00', currency: 'LKR', createdAt: '2026-09-01T12:00:00Z' }], hasMore: false }),
  listEventRegistrations: vi.fn().mockResolvedValue({ items: [{ bookingId: 'booking-b', accountId: 'member-b', nickname: 'Nimal', email: 'nimal@example.test', eventId: 'event-a', state: 'PENDING_PAYMENT', paymentMethod: 'ONLINE', amount: '2500.00', currency: 'LKR', createdAt: '2026-09-02T12:00:00Z' }], hasMore: false }),
}))

describe('RegistrationsView', () => {
  it('loads the all-event feed and narrows to one event', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const events = [{ eventId: 'event-a', name: 'Seaside Social' }] as ManagedEvent[]
    const onSelectedEventIdChange = vi.fn()
    const { rerender } = render(<QueryClientProvider client={client}><RegistrationsView events={events} eventsLoading={false} eventsError={false} selectedEventId="" onSelectedEventIdChange={onSelectedEventIdChange} /></QueryClientProvider>)
    expect(await screen.findByText('Asha')).toBeVisible()
    expect(screen.getByText('Not collected in app')).toBeVisible()
    expect(listAllRegistrations).toHaveBeenCalledWith(0)

    fireEvent.change(screen.getByLabelText('Event'), { target: { value: 'event-a' } })
    expect(onSelectedEventIdChange).toHaveBeenCalledWith('event-a')
    rerender(<QueryClientProvider client={client}><RegistrationsView events={events} eventsLoading={false} eventsError={false} selectedEventId="event-a" onSelectedEventIdChange={onSelectedEventIdChange} /></QueryClientProvider>)
    expect(await screen.findByText('Nimal')).toBeVisible()
    expect(listEventRegistrations).toHaveBeenCalledWith('event-a', 0)
  })
})
