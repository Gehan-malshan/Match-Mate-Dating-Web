import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { LandingPage } from './LandingPage'

describe('LandingPage', () => {
  it('presents the privacy-first, no-chat, and no-ML product boundaries', () => {
    render(<LandingPage />)

    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
      'Where mysterymeets connection.',
    )
    expect(screen.getByText(/no machine-learning model/i)).toBeVisible()
    expect(screen.getByText(/no member-to-member chat/i)).toBeVisible()
    expect(screen.getByText(/registration opens after identity/i)).toBeVisible()
    expect(screen.getByText(/server-confirmed booking unlocks eligibility/i)).toBeVisible()
  })

  it('provides navigation to the redesigned landing-page chapters', () => {
    render(<LandingPage />)

    const navigation = within(
      screen.getByRole('navigation', { name: 'Primary navigation' }),
    )

    expect(navigation.getByRole('link', { name: 'Why MatchMate' })).toHaveAttribute(
      'href',
      '#trust',
    )
    expect(navigation.getByRole('link', { name: 'The journey' })).toHaveAttribute(
      'href',
      '#how-it-works',
    )
    expect(navigation.getByRole('link', { name: 'Events' })).toHaveAttribute(
      'href',
      '#events',
    )
  })

  it('lets keyboard and pointer users expand each journey chapter', () => {
    render(<LandingPage />)

    const firstStep = screen.getByRole('button', { name: /begin privately/i })
    const secondStep = screen.getByRole('button', { name: /choose the room/i })

    expect(firstStep).toHaveAttribute('aria-expanded', 'true')
    expect(secondStep).toHaveAttribute('aria-expanded', 'false')

    fireEvent.click(secondStep)

    expect(firstStep).toHaveAttribute('aria-expanded', 'false')
    expect(secondStep).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText(/dates, broad location, format, eligibility/i)).toBeVisible()
  })

  it('uses the approved MatchMate logo in the header and footer', () => {
    const { container } = render(<LandingPage />)

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
})
