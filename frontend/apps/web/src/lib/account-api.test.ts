import { afterEach, describe, expect, it, vi } from 'vitest'
import { blockMember, getCommunityProfile, listCommunityProfiles } from './account-api'

afterEach(() => vi.unstubAllGlobals())

describe('community profile API client', () => {
  it('loads a bounded community page and encodes its cursor', async () => {
    const payload = { items: [], nextCursor: 'next' }
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(payload), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)
    await expect(listCommunityProfiles('cursor value')).resolves.toEqual(payload)
    expect(fetchMock.mock.calls[0][0]).toContain('/community/profiles?limit=8&cursor=cursor%20value')
  })

  it('encodes profile identifiers for detail requests', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ profileId: 'safe' }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    await getCommunityProfile('id/with spaces')
    expect(fetchMock.mock.calls[0][0]).toContain('/community/profiles/id%2Fwith%20spaces')
  })

  it('uses the block endpoint without exposing profile details', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)
    await blockMember('profile-1')
    expect(fetchMock.mock.calls[0][0]).toContain('/users/me/blocks')
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: 'POST', body: JSON.stringify({ accountId: 'profile-1' }) })
  })
})
