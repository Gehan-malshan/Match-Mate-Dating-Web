import { afterEach, describe, expect, it, vi } from 'vitest'
import { generateMatchingRun, login, refresh } from './api'

afterEach(() => vi.unstubAllGlobals())

describe('administrator authentication', () => {
  it('rejects a valid organizer without administrator authority', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ accessToken: 'token' }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ account: { id: 'organizer', roles: ['member', 'organizer'] } }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    await expect(login('organizer@example.test', 'password')).rejects.toMatchObject({ code: 'ADMIN_ROLE_REQUIRED' })
  })

  it('accepts an administrator account', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ accessToken: 'token' }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ account: { id: 'admin', roles: ['member', 'admin'] } }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    await expect(login('admin@example.test', 'password')).resolves.toMatchObject({ id: 'admin' })
  })

  it('shares one rotating refresh request across concurrent session restores', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ accessToken: 'refreshed-token' }), { status: 200 }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(Promise.all([refresh(), refresh()])).resolves.toEqual([true, true])
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })
})

describe('matchmaking commands', () => {
  it('adds an idempotency key when generating a run', async () => {
    vi.stubGlobal('crypto', { randomUUID: () => 'fixed-command-id' })
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ runId: 'run-1' }), { status: 201 }))
    vi.stubGlobal('fetch', fetchMock)
    await generateMatchingRun('event-1')
    const request = fetchMock.mock.calls[0]
    expect(request[0]).toContain('/events/event-1/matching-runs')
    expect(new Headers(request[1].headers).get('Idempotency-Key')).toBe('generate-fixed-command-id')
  })
})
