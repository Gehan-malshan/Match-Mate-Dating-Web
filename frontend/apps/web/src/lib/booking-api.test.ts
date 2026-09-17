import {afterEach,describe,expect,it,vi} from 'vitest'
import {createBooking,findResumableBooking,initiatePayment,type Booking} from './booking-api'
const response=(data:unknown)=>new Response(JSON.stringify({data}),{status:200,headers:{'Content-Type':'application/json'}})
const booking=(eventId:string,state:Booking['state']):Booking=>({bookingId:'booking-'+eventId,eventId,state,amount:'2500.00',currency:'LKR',policyVersion:1,expiresAt:'2026-08-30T12:00:00Z',version:1,createdAt:'2026-08-30T11:45:00Z'})
afterEach(()=>vi.unstubAllGlobals())
describe('GraphQL booking and payment client',()=>{
  it('creates a hold using only event and idempotency identifiers',async()=>{const fetchMock=vi.fn().mockResolvedValue(response({createBooking:booking('e1','PENDING_PAYMENT')}));vi.stubGlobal('fetch',fetchMock);await createBooking('e1','key-1');const body=JSON.parse(fetchMock.mock.calls[0][1].body);expect(body.variables).toEqual({eventId:'e1',idempotencyKey:'key-1'});expect(body.variables).not.toHaveProperty('amount')})
  it('does not send an amount during payment initiation',async()=>{const checkout={payment:{},actionUrl:'https://sandbox.payhere.lk/pay/checkout',fields:{}};const fetchMock=vi.fn().mockResolvedValue(response({initiatePayment:checkout}));vi.stubGlobal('fetch',fetchMock);await initiatePayment({bookingId:'b1',firstName:'A',lastName:'B',email:'a@example.test',phone:'0700000000',address:'Test road',city:'Colombo',country:'Sri Lanka'},'key-2');const body=JSON.parse(fetchMock.mock.calls[0][1].body);expect(body.variables.input).not.toHaveProperty('amount');expect(body.variables.idempotencyKey).toBe('key-2')})
  it('recovers an existing pending hold after revisiting',()=>{expect(findResumableBooking([booking('event-1','CONFIRMED'),booking('event-2','PENDING_PAYMENT')],'event-2')?.bookingId).toBe('booking-event-2');expect(findResumableBooking([booking('event-1','EXPIRED')],'event-1')).toBeNull()})
})
