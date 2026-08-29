import {authenticatedRequest} from './account-api'

const bookingBase=import.meta.env.VITE_BOOKING_API_URL??'http://localhost:8085/api/v1'
const paymentBase=import.meta.env.VITE_PAYMENT_API_URL??'http://localhost:8084/api/v1'
export type BookingState='PENDING_PAYMENT'|'CONFIRMED'|'EXPIRED'|'CANCELLED'|'PAYMENT_REVIEW'
export type Booking={bookingId:string;eventId:string;state:BookingState;amount:string;currency:string;policyVersion:number;expiresAt:string;version:number;createdAt:string;confirmedAt?:string;cancelledAt?:string}
export type PaymentState='PENDING'|'COMPLETED'|'FAILED'|'REVIEW'
export type Payment={paymentId:string;bookingId:string;orderId:string;amount:string;currency:string;provider:'PAYHERE';state:PaymentState;version:number;createdAt:string;updatedAt:string;completedAt?:string}
export type Checkout={payment:Payment;actionUrl:string;fields:Record<string,string>}
export const createBooking=(eventId:string,key=crypto.randomUUID())=>authenticatedRequest<Booking>(bookingBase,'/bookings',{method:'POST',headers:{'Idempotency-Key':key},body:JSON.stringify({eventId})})
export const listBookings=()=>authenticatedRequest<{items:Booking[]}>(bookingBase,'/bookings')
export const cancelBooking=(bookingId:string,key=crypto.randomUUID())=>authenticatedRequest<Booking>(bookingBase,`/bookings/${encodeURIComponent(bookingId)}/cancel`,{method:'POST',headers:{'Idempotency-Key':key}})
export const getPayment=(bookingId:string)=>authenticatedRequest<Payment>(paymentBase,`/bookings/${encodeURIComponent(bookingId)}/payment`)
export const initiatePayment=(input:{bookingId:string;firstName:string;lastName:string;email:string;phone:string;address:string;city:string;country:string},key=crypto.randomUUID())=>authenticatedRequest<Checkout>(paymentBase,'/payments/initiate',{method:'POST',headers:{'Idempotency-Key':key},body:JSON.stringify(input)})
export function submitCheckout(checkout:Checkout){const form=document.createElement('form');form.method='POST';form.action=checkout.actionUrl;for(const [name,value] of Object.entries(checkout.fields)){const input=document.createElement('input');input.type='hidden';input.name=name;input.value=value;form.append(input)}document.body.append(form);form.submit()}
