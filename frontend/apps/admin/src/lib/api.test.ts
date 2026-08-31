import {afterEach,describe,expect,it,vi} from 'vitest'
import {generateMatchingRun,login,refresh} from './api'
const response=(data:unknown)=>new Response(JSON.stringify({data}),{status:200,headers:{'Content-Type':'application/json'}})
afterEach(()=>vi.unstubAllGlobals())
describe('administrator GraphQL client',()=>{
  it('rejects a non-admin role returned by login',async()=>{vi.stubGlobal('fetch',vi.fn().mockResolvedValue(response({login:{accessToken:'token',me:{account:{id:'organizer',roles:['member','organizer']}}}})));await expect(login('organizer@example.test','password')).rejects.toMatchObject({code:'ADMIN_ROLE_REQUIRED'})})
  it('accepts an administrator role',async()=>{vi.stubGlobal('fetch',vi.fn().mockResolvedValue(response({login:{accessToken:'token',me:{account:{id:'admin',roles:['member','admin']}}}})));await expect(login('admin@example.test','password')).resolves.toMatchObject({id:'admin'})})
  it('shares one rotating refresh request',async()=>{const fetchMock=vi.fn().mockResolvedValue(response({refreshSession:{accessToken:'refreshed-token'}}));vi.stubGlobal('fetch',fetchMock);await expect(Promise.all([refresh(),refresh()])).resolves.toEqual([true,true]);expect(fetchMock).toHaveBeenCalledTimes(1)})
  it('passes matching idempotency as a GraphQL variable',async()=>{vi.stubGlobal('crypto',{randomUUID:()=> 'fixed-command-id'});const fetchMock=vi.fn().mockResolvedValue(response({generateMatchingRun:{runId:'run-1'}}));vi.stubGlobal('fetch',fetchMock);await generateMatchingRun('event-1');const body=JSON.parse(fetchMock.mock.calls[0][1].body);expect(body.variables).toEqual({eventId:'event-1',key:'generate-fixed-command-id'})})
})
