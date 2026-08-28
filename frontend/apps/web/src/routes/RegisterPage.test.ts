import { describe, expect, it } from 'vitest'
import { toRegistrationRequest } from './RegisterPage'

describe('registration API payload', () => {
  it('does not send the UI-only consent checkbox as an unknown API field', () => {
    const request = toRegistrationRequest({
      email: 'member@example.com',
      password: 'MatchMateTest123!',
      nickname: 'Member',
      dateOfBirth: '1995-01-01',
      consent: true,
    })

    expect(request).toEqual({
      email: 'member@example.com',
      password: 'MatchMateTest123!',
      nickname: 'Member',
      dateOfBirth: '1995-01-01',
      consentVersion: 'privacy-2026-08',
    })
    expect(request).not.toHaveProperty('consent')
  })
})
