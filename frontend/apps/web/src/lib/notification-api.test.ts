import {afterEach,describe,expect,it,vi} from 'vitest'
import {listNotifications,markAllNotificationsRead,markNotificationRead} from './notification-api'
const response=(data:unknown)=>new Response(JSON.stringify({data}),{status:200,headers:{'Content-Type':'application/json'}})
afterEach(()=>vi.unstubAllGlobals())
describe('GraphQL notification client',()=>{
  it('loads bounded member pagination',async()=>{const fetchMock=vi.fn().mockResolvedValue(response({notifications:{items:[],unreadCount:0}}));vi.stubGlobal('fetch',fetchMock);await listNotifications(5,'next cursor');const [,init]=fetchMock.mock.calls[0];expect(JSON.parse(init.body).variables).toEqual({limit:5,cursor:'next cursor'});expect(init.credentials).toBe('include')})
  it('uses explicit mutations without account identifiers',async()=>{const fetchMock=vi.fn().mockResolvedValueOnce(response({markNotificationRead:{success:true}})).mockResolvedValueOnce(response({markAllNotificationsRead:{updatedCount:2}}));vi.stubGlobal('fetch',fetchMock);await markNotificationRead('notification-1');await markAllNotificationsRead();const bodies=fetchMock.mock.calls.map((call)=>String(call[1].body));expect(bodies[0]).toContain('markNotificationRead');expect(bodies[1]).toContain('markAllNotificationsRead');expect(bodies.join(' ')).not.toContain('accountId')})
})
