import {afterEach,describe,expect,it,vi} from 'vitest'
import {createBooking,initiatePayment} from './booking-api'

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
})
