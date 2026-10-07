import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { LandingPage } from './LandingPage'

vi.mock('../lib/event-api', () => ({
  eventImageUrl: () => undefined,
  listEvents: vi.fn().mockResolvedValue({
    items: [{
      eventId: 'preview-event-1',
      name: 'Preview Event',
      description: 'A public preview event for this page.',
      broadLocation: 'Colombo 03',
      timeZone: 'Asia/Colombo',
      startsAt: '2026-11-04T12:30:00Z',
      endsAt: '2026-11-04T15:30:00Z',
      registrationOpensAt: '2026-10-21T12:30:00Z',
      registrationClosesAt: '2026-11-03T12:30:00Z',
      price: '3000.00',
      currency: 'LKR',
      configuredCapacity: 36,
      matchingRulesetVersion: 'development-v1',
      status: 'PUBLISHED',
      version: 1,
    }],
    limit: 12,
  }),
}))

function renderLandingPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={queryClient}><LandingPage /></QueryClientProvider>)
}

describe('LandingPage', () => {
  it('presents the privacy-first, no-chat, and no-ML product boundaries', () => {
    renderLandingPage()

    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
      'Less swiping.More meaningful meetings.',
    )
    expect(screen.getByText(/not inferred traits or machine learning/i)).toBeVisible()
    expect(screen.getByText(/no member-to-member chat/i)).toBeVisible()
    expect(screen.getByText(/confirm your booking securely/i)).toBeVisible()
  })

  it('provides navigation to the redesigned landing-page chapters', () => {
    renderLandingPage()

    const navigation = within(
      screen.getByRole('navigation', { name: 'Primary navigation' }),
    )

    expect(navigation.getByRole('link', { name: 'About' })).toHaveAttribute(
      'href',
      '#about',
    )
    expect(navigation.getByRole('link', { name: 'How it works' })).toHaveAttribute(
      'href',
      '#how-it-works',
    )
    expect(navigation.getByRole('link', { name: 'Events' })).toHaveAttribute(
      'href',
      '#events',
    )
  })

  it('shows the full matchmaking journey without hiding steps behind controls', () => {
    renderLandingPage()

    const process = screen.getByRole('list', { name: 'How MatchMate works' })
    expect(within(process).getAllByRole('listitem')).toHaveLength(5)
    expect(within(process).getByText(/reserve an event place/i)).toBeVisible()
    expect(within(process).getByText(/connect by mutual choice/i)).toBeVisible()
  })

  it('explains the complete five-step matchmaking process in plain language', () => {
    renderLandingPage()

    expect(screen.getByText(/preferences, deal-breakers, and shared values/i)).toBeInTheDocument()
    expect(screen.getByText(/unless both people choose to continue/i)).toBeInTheDocument()
  })

  it('uses the approved MatchMate logo in the header and footer', () => {
    const { container } = renderLandingPage()

    const logoMarks = container.querySelectorAll<HTMLImageElement>(
      'img.brand-mark[src="/brand/matchmate-logo-nav.png"]',
    )

    expect(logoMarks).toHaveLength(2)
    expect(screen.getByRole('link', { name: 'MatchMate home' })).toContainElement(
      logoMarks[0],
    )
    expect(screen.getByRole('link', { name: 'Back to the top' })).toContainElement(
      logoMarks[1],
    )
  })

  it('uses distinct local images for the About, process, and event chapters', async () => {
    const { container } = renderLandingPage()

    expect(container.querySelector('img[src="/images/matchmate-about-cafe.png"]')).toBeInTheDocument()
    expect(container.querySelector('img[src="/images/matchmate-event-checkin.png"]')).toBeInTheDocument()
    await waitFor(() => expect(container.querySelector('img[src="/images/matchmate-rooftop-event.png"]')).toBeInTheDocument())
  })

  it('uses the generated happy-couple artwork as the hero background', () => {
    renderLandingPage()

    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Less swiping.More meaningful meetings.')
    expect(screen.queryByLabelText('MatchMate privacy principle')).not.toBeInTheDocument()
  })

  it('provides structured footer navigation without inventing unavailable pages', () => {
    renderLandingPage()

    expect(screen.getByRole('navigation', { name: 'Footer navigation' })).toBeVisible()
    expect(screen.getByRole('navigation', { name: 'Trust and safety navigation' })).toBeVisible()
    expect(screen.getByText(/colombo event announcements/i)).toBeVisible()
    expect(screen.getByRole('link', { name: /back to top/i })).toHaveAttribute('href', '#top')
  })

  it('links the event action to implemented event discovery', () => {
    renderLandingPage()

    expect(screen.getByRole('link', { name: /view more events/i })).toHaveAttribute('href', '/events')
  })
})
