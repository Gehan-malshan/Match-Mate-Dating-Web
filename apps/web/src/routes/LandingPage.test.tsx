import { render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { LandingPage } from './LandingPage'

describe('LandingPage', () => {
  it('presents the privacy-first, no-ML product boundaries', () => {
    render(<LandingPage />)

    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
      'Where mysterymeets connection.',
    )
    expect(screen.getByText(/does not use machine learning/i)).toBeVisible()
    expect(screen.getByText(/without member-to-member chat/i)).toBeVisible()
    expect(screen.getByText(/registration and login will open/i)).toBeVisible()
  })

  it('provides navigation to the main landing-page sections', () => {
    render(<LandingPage />)

    const navigation = within(
      screen.getByRole('navigation', { name: 'Primary navigation' }),
    )

    expect(navigation.getByRole('link', { name: 'How it works' })).toHaveAttribute(
      'href',
      '#how-it-works',
    )
    expect(navigation.getByRole('link', { name: 'Events' })).toHaveAttribute(
      'href',
      '#events',
    )
    expect(navigation.getByRole('link', { name: 'About' })).toHaveAttribute(
      'href',
      '#about',
    )
  })
})
