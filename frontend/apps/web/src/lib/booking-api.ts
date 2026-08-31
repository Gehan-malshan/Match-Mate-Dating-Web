import {graphqlClient} from './account-api'

export type BookingState='PENDING_PAYMENT'|'CONFIRMED'|'EXPIRED'|'CANCELLED'|'PAYMENT_REVIEW'
export type Booking={bookingId:string;eventId:string;state:BookingState;amount:string;currency:string;policyVersion:number;expiresAt:string;version:number;createdAt:string;confirmedAt?:string;cancelledAt?:string}
export type PaymentState='PENDING'|'COMPLETED'|'FAILED'|'REVIEW'
export type Payment={paymentId:string;bookingId:string;orderId:string;amount:string;currency:string;provider:'PAYHERE';state:PaymentState;version:number;createdAt:string;updatedAt:string;completedAt?:string}
export type Checkout={payment:Payment;actionUrl:string;fields:Record<string,string>}
const bookingFields='bookingId eventId state amount currency policyVersion expiresAt version createdAt confirmedAt cancelledAt'
const paymentFields='paymentId bookingId orderId amount currency provider state version createdAt updatedAt completedAt'
export async function createBooking(eventId:string,idempotencyKey:string=crypto.randomUUID()){const data=await graphqlClient.execute<{createBooking:Booking}>(`mutation CreateBooking($eventId:ID!,$idempotencyKey:String!){createBooking(eventId:$eventId,idempotencyKey:$idempotencyKey){${bookingFields}}}`,{eventId,idempotencyKey});return data.createBooking}
export async function listBookings(){const data=await graphqlClient.execute<{bookings:{items:Booking[]}}>(`query Bookings{bookings{items{${bookingFields}}}}`);return data.bookings}
export async function cancelBooking(bookingId:string,idempotencyKey:string=crypto.randomUUID()){const data=await graphqlClient.execute<{cancelBooking:Booking}>(`mutation CancelBooking($bookingId:ID!,$idempotencyKey:String!){cancelBooking(bookingId:$bookingId,idempotencyKey:$idempotencyKey){${bookingFields}}}`,{bookingId,idempotencyKey});return data.cancelBooking}
export async function getPayment(bookingId:string){const data=await graphqlClient.execute<{payment:Payment}>(`query Payment($bookingId:ID!){payment(bookingId:$bookingId){${paymentFields}}}`,{bookingId});return data.payment}
export async function initiatePayment(input:{bookingId:string;firstName:string;lastName:string;email:string;phone:string;address:string;city:string;country:string},idempotencyKey:string=crypto.randomUUID()){const data=await graphqlClient.execute<{initiatePayment:Checkout}>(`mutation InitiatePayment($input:CheckoutCustomerInput!,$idempotencyKey:String!){initiatePayment(input:$input,idempotencyKey:$idempotencyKey){payment{${paymentFields}} actionUrl fields}}`,{input,idempotencyKey});return data.initiatePayment}
export function submitCheckout(checkout:Checkout){const form=document.createElement('form');form.method='POST';form.action=checkout.actionUrl;for(const [name,value] of Object.entries(checkout.fields)){const input=document.createElement('input');input.type='hidden';input.name=name;input.value=value;form.append(input)}document.body.append(form);form.submit()}
export const findResumableBooking=(items:Booking[]|undefined,eventId:string)=>items?.find((item)=>item.eventId===eventId&&item.state==='PENDING_PAYMENT')??null
