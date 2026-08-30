import {afterEach,describe,expect,it,vi} from 'vitest'
import {createBooking,findResumableBooking,initiatePayment,type Booking} from './booking-api'

const booking=(eventId:string,state:Booking['state']):Booking=>({bookingId:'booking-'+eventId,eventId,state,amount:'2500.00',currency:'LKR',policyVersion:1,expiresAt:'2026-08-30T12:00:00Z',version:1,createdAt:'2026-08-30T11:45:00Z'})

afterEach(()=>vi.unstubAllGlobals())

describe('booking and payment client',()=>{
  it('creates a hold using only the event identifier',async()=>{
    const fetchMock=vi.fn().mockResolvedValue(new Response(JSON.stringify({bookingId:'b1',eventId:'e1',state:'PENDING_PAYMENT'}),{status:201,headers:{'Content-Type':'application/json'}}))
    vi.stubGlobal('fetch',fetchMock)
    await createBooking('e1','11111111-1111-4111-8111-111111111111')
    const [,init]=fetchMock.mock.calls[0]
    expect(JSON.parse(String(init.body))).toEqual({eventId:'e1'})
    expect(init.headers.get('Idempotency-Key')).toBeTruthy()
  })

  it('does not send an amount during payment initiation',async()=>{
    const fetchMock=vi.fn().mockResolvedValue(new Response(JSON.stringify({payment:{},actionUrl:'https://sandbox.payhere.lk/pay/checkout',fields:{}}),{status:201,headers:{'Content-Type':'application/json'}}))
    vi.stubGlobal('fetch',fetchMock)
    await initiatePayment({bookingId:'b1',firstName:'A',lastName:'B',email:'a@example.test',phone:'0700000000',address:'Test road',city:'Colombo',country:'Sri Lanka'},'22222222-2222-4222-8222-222222222222')
    const [,init]=fetchMock.mock.calls[0]
    expect(JSON.parse(String(init.body))).not.toHaveProperty('amount')
  })

  it('recovers an existing pending hold after the event page is revisited',()=>{
    const recovered=findResumableBooking([booking('event-1','CONFIRMED'),booking('event-2','PENDING_PAYMENT')],'event-2')
    expect(recovered?.bookingId).toBe('booking-event-2')
    expect(findResumableBooking([booking('event-1','EXPIRED'),booking('event-1','CANCELLED')],'event-1')).toBeNull()
  })
})
