import {useMutation,useQuery,useQueryClient} from '@tanstack/react-query'
import {Link} from '@tanstack/react-router'
import {MemberNavigation} from '../components/MemberNavigation'
import {cancelBooking,getPayment,listBookings} from '../lib/booking-api'
import './BookingsPage.css'

const labels={PENDING_PAYMENT:'Awaiting payment',CONFIRMED:'Confirmed',EXPIRED:'Hold expired',CANCELLED:'Cancelled',PAYMENT_REVIEW:'Manual review'} as const

function PaymentStatus({bookingId}:{bookingId:string}){
  const query=useQuery({queryKey:['payment',bookingId],queryFn:()=>getPayment(bookingId),retry:false,refetchInterval:(q)=>q.state.data?.state==='PENDING'?5000:false})
  if(query.isPending)return <span>Checking payment…</span>
  if(query.isError)return <span>No payment started</span>
  return <span>Payment: {query.data.state.toLowerCase()}</span>
}

export function BookingsPage(){
  const client=useQueryClient()
  const query=useQuery({queryKey:['bookings'],queryFn:listBookings,refetchInterval:10000})
  const cancel=useMutation({mutationFn:(bookingId:string)=>cancelBooking(bookingId),onSuccess:()=>client.invalidateQueries({queryKey:['bookings']})})
  return <main className="bookings-page"><MemberNavigation active="bookings"/><section className="bookings-shell"><p className="event-detail-kicker">Your event journey</p><h1>My bookings</h1><p className="bookings-lead">Payment redirects are not proof of success. This page reflects Booking and Payment server state.</p>{query.isPending&&<p role="status">Loading your bookings…</p>}{query.isError&&<div className="bookings-empty"><h2>Sign in to view bookings</h2><Link className="button" to="/login">Log in</Link></div>}{query.data?.items.length===0&&<div className="bookings-empty"><h2>No bookings yet</h2><p>Explore an event and reserve a seat when registration is open.</p><Link className="button" to="/events">Explore events</Link></div>}<div className="booking-list">{query.data?.items.map((booking)=><article key={booking.bookingId}><div><span className={`booking-state state-${booking.state.toLowerCase()}`}>{labels[booking.state]}</span><h2>{booking.currency} {booking.amount}</h2><p>Event reference {booking.eventId}</p><small>Created {new Date(booking.createdAt).toLocaleString('en-LK')}</small></div><div className="booking-actions"><PaymentStatus bookingId={booking.bookingId}/><Link to="/events/$eventId" params={{eventId:booking.eventId}}>{booking.state==='PENDING_PAYMENT'?'Complete payment':'View event'}</Link>{booking.state==='PENDING_PAYMENT'&&<button type="button" disabled={cancel.isPending} onClick={()=>cancel.mutate(booking.bookingId)}>Cancel hold</button>}</div></article>)}</div></section></main>
}
