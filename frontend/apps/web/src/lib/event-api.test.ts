import {afterEach,describe,expect,it,vi} from 'vitest'
import {eventImageUrl,getEvent,listEvents} from './event-api'
const response=(data:unknown)=>new Response(JSON.stringify({data}),{status:200,headers:{'Content-Type':'application/json'}})
afterEach(()=>vi.unstubAllGlobals())
describe('GraphQL event client',()=>{
  it('reads the bounded public event page',async()=>{const payload={items:[],limit:12};const fetchMock=vi.fn().mockResolvedValue(response({events:payload}));vi.stubGlobal('fetch',fetchMock);await expect(listEvents()).resolves.toEqual(payload);const body=JSON.parse(fetchMock.mock.calls[0][1].body);expect(body.query).toContain('events(limit:12');expect(fetchMock.mock.calls[0][1].credentials).toBe('include')})
  it('passes event identifiers as GraphQL variables',async()=>{const fetchMock=vi.fn().mockResolvedValue(response({event:{eventId:'safe'}}));vi.stubGlobal('fetch',fetchMock);await getEvent('id/with spaces');expect(JSON.parse(fetchMock.mock.calls[0][1].body).variables.eventId).toBe('id/with spaces')})
  it('uses published event artwork only when one is attached',()=>{expect(eventImageUrl({eventId:'abc',imageVersion:0})).toBeUndefined();expect(eventImageUrl({eventId:'abc',imageVersion:2})).toContain('/media/events/abc?v=2')})
})
