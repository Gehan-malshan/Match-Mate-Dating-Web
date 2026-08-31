import {useState} from 'react'
import {useMutation,useQuery,useQueryClient} from '@tanstack/react-query'
import {Link} from '@tanstack/react-router'
import {createBooking,findResumableBooking,initiatePayment,listBookings,submitCheckout,type Booking} from '../lib/booking-api'

export function BookingPurchasePanel({eventId,isOpen}:{eventId:string;isOpen:boolean}){
  const [booking,setBooking]=useState<Booking|null>(null)
  const [message,setMessage]=useState('')
  const client=useQueryClient()
  const bookings=useQuery({queryKey:['bookings'],queryFn:listBookings,retry:false,staleTime:5000})
  const activeBooking=booking??findResumableBooking(bookings.data?.items,eventId)
  const hold=useMutation({mutationFn:()=>createBooking(eventId),onSuccess:(value)=>{setBooking(value);setMessage('Your seat is held. Complete checkout before the timer expires.');void client.invalidateQueries({queryKey:['bookings']})},onError:()=>{setMessage('We could not reserve this seat. Sign in, check eligibility, or try again.');void client.invalidateQueries({queryKey:['bookings']})}})
  const checkout=useMutation({mutationFn:(input:{firstName:string;lastName:string;email:string;phone:string;address:string;city:string;country:string})=>initiatePayment({bookingId:activeBooking!.bookingId,...input}),onSuccess:submitCheckout,onError:()=>setMessage('Checkout could not start. Your hold remains visible in My bookings.')})
  if(!isOpen&&!activeBooking)return <p className="purchase-message">Registration is not currently open.</p>
  if(bookings.isPending&&!booking)return <button className="button event-detail-action" type="button" disabled>Checking your booking…</button>
  if(!activeBooking)return <><button className="button event-detail-action" type="button" disabled={hold.isPending} onClick={()=>hold.mutate()}>{hold.isPending?'Reserving…':'Reserve a seat'} <b aria-hidden="true">→</b></button>{message&&<p className="purchase-message" role="status">{message} <Link to="/login">Sign in</Link></p>}</>
  const checkoutMessage=message||'Your existing seat hold is ready. Complete payment before it expires.'
  return <div className="purchase-flow"><p className="purchase-message" role="status">{checkoutMessage}</p><p className="hold-reference">Hold expires <strong>{new Date(activeBooking.expiresAt).toLocaleTimeString('en-LK',{hour:'2-digit',minute:'2-digit'})}</strong></p><form onSubmit={(event)=>{event.preventDefault();const data=new FormData(event.currentTarget);checkout.mutate({firstName:String(data.get('firstName')),lastName:String(data.get('lastName')),email:String(data.get('email')),phone:String(data.get('phone')),address:String(data.get('address')),city:String(data.get('city')),country:String(data.get('country'))})}}><div className="purchase-name-row"><label>First name<input name="firstName" autoComplete="given-name" required /></label><label>Last name<input name="lastName" autoComplete="family-name" required /></label></div><label>Email<input name="email" type="email" autoComplete="email" required /></label><label>Phone<input name="phone" type="tel" autoComplete="tel" required /></label><label>Billing address<input name="address" autoComplete="street-address" required /></label><div className="purchase-name-row"><label>City<input name="city" autoComplete="address-level2" required /></label><label>Country<input name="country" autoComplete="country-name" defaultValue="Sri Lanka" required /></label></div><button className="button event-detail-action" disabled={checkout.isPending}>{checkout.isPending?'Opening PayHere…':'Pay '+activeBooking.currency+' '+activeBooking.amount}</button></form><p className="event-detail-ticket-note">PayHere receives these checkout details transiently. MatchMate confirms payment only after a verified server callback.</p></div>
}
