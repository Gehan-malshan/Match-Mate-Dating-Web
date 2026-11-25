import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { EventsView } from '../../../admin/src/views/EventsView'
import { archiveEvent, type ManagedEvent } from '../../../admin/src/lib/api'

vi.mock('../../../admin/src/lib/api', () => ({
  archiveEvent: vi.fn().mockResolvedValue({}),
  restoreEvent: vi.fn().mockResolvedValue({}),
  listArchivedEvents: vi.fn().mockResolvedValue({ items: [] }),
}))

afterEach(() => vi.restoreAllMocks())

describe('Event manager Delete', () => {
  it('asks for confirmation before archiving a draft', async () => {
    const event = { eventId: 'event-a', name: 'Duplicate Draft', status: 'DRAFT', startsAt: '2026-11-01T10:00:00Z', timeZone: 'Asia/Colombo', broadLocation: 'Colombo', configuredCapacity: 20, currency: 'LKR', price: '2500.00', paymentOptions: 'AT_VENUE', imageVersion: 0, version: 2, description: '' } as ManagedEvent
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={client}><EventsView account={{ id: 'admin', roles: ['admin'] }} events={[event]} loading={false} error={false} onViewRegistrations={() => {}} /></QueryClientProvider>)
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(archiveEvent).toHaveBeenCalledWith(event))
  })
})
