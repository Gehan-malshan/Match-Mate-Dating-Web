import {afterEach,describe,expect,it,vi} from 'vitest'
import {blockMember,getCommunityProfile,listCommunityProfiles,refresh} from './account-api'
const response=(data:unknown)=>new Response(JSON.stringify({data}),{status:200,headers:{'Content-Type':'application/json'}})
afterEach(()=>vi.unstubAllGlobals())
describe('GraphQL account client',()=>{
  it('passes community pagination as GraphQL variables',async()=>{const payload={items:[],nextCursor:'next'};const fetchMock=vi.fn().mockResolvedValue(response({communityProfiles:payload}));vi.stubGlobal('fetch',fetchMock);await expect(listCommunityProfiles('cursor value')).resolves.toEqual(payload);const body=JSON.parse(fetchMock.mock.calls[0][1].body);expect(body.variables).toEqual({cursor:'cursor value'});expect(body.query).toContain('communityProfiles(limit:8')})
  it('passes opaque profile identifiers as variables',async()=>{const fetchMock=vi.fn().mockResolvedValue(response({communityProfile:{profileId:'safe'}}));vi.stubGlobal('fetch',fetchMock);await getCommunityProfile('id/with spaces');expect(JSON.parse(fetchMock.mock.calls[0][1].body).variables.profileId).toBe('id/with spaces')})
  it('blocks by identifier without exposing profile details',async()=>{const fetchMock=vi.fn().mockResolvedValue(response({blockMember:{success:true}}));vi.stubGlobal('fetch',fetchMock);await blockMember('profile-1');const body=JSON.parse(fetchMock.mock.calls[0][1].body);expect(body.variables).toEqual({accountId:'profile-1'});expect(body.query).toContain('blockMember')})
  it('shares one rotating refresh request',async()=>{const fetchMock=vi.fn().mockResolvedValue(response({refreshSession:{accessToken:'refreshed-token'}}));vi.stubGlobal('fetch',fetchMock);await expect(Promise.all([refresh(),refresh()])).resolves.toEqual([true,true]);expect(fetchMock).toHaveBeenCalledTimes(1)})
})
