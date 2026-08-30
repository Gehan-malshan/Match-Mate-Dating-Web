import {afterEach,describe,expect,it,vi} from 'vitest'
import {listNotifications,markAllNotificationsRead,markNotificationRead} from './notification-api'

afterEach(()=>vi.unstubAllGlobals())

describe('notification client',()=>{
  it('loads only the authenticated member feed with bounded pagination',async()=>{
    const fetchMock=vi.fn().mockResolvedValue(new Response(JSON.stringify({items:[],unreadCount:0}),{status:200,headers:{'Content-Type':'application/json'}}))
    vi.stubGlobal('fetch',fetchMock)
    await listNotifications(5,'next cursor')
    expect(fetchMock.mock.calls[0][0]).toContain('/notifications?limit=5&cursor=next%20cursor')
    expect(fetchMock.mock.calls[0][1].credentials).toBe('include')
  })

  it('uses explicit read commands without sending account identifiers',async()=>{
    const fetchMock=vi.fn()
      .mockResolvedValueOnce(new Response(null,{status:204}))
      .mockResolvedValueOnce(new Response(JSON.stringify({updatedCount:2}),{status:200,headers:{'Content-Type':'application/json'}}))
    vi.stubGlobal('fetch',fetchMock)
    await markNotificationRead('notification-1')
    await markAllNotificationsRead()
    expect(fetchMock.mock.calls[0][0]).toContain('/notifications/notification-1/read')
    expect(fetchMock.mock.calls[0][1].method).toBe('PATCH')
    expect(fetchMock.mock.calls[1][1].method).toBe('POST')
    expect(fetchMock.mock.calls.flat().join(' ')).not.toContain('accountId')
  })
})
