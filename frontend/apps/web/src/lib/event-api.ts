const eventBaseUrl=import.meta.env.VITE_EVENT_API_URL??'http://localhost:8082/api/v1'
export type EventStatus='PUBLISHED'|'REGISTRATION_OPEN'|'REGISTRATION_CLOSED'
export type PublicEvent={eventId:string;name:string;description:string;broadLocation:string;timeZone:string;startsAt:string;endsAt:string;registrationOpensAt:string;registrationClosesAt:string;price:string;currency:string;configuredCapacity:number;matchingRulesetVersion:string;status:EventStatus;version:number}
export type EventPage={items:PublicEvent[];nextCursor?:string;limit:number}
async function read<T>(path:string):Promise<T>{const response=await fetch(`${eventBaseUrl}${path}`,{headers:{Accept:'application/json'}});if(!response.ok)throw await response.json();return response.json() as Promise<T>}
export const listEvents=(cursor='')=>read<EventPage>(`/events?limit=12${cursor?`&cursor=${encodeURIComponent(cursor)}`:''}`)
export const getEvent=(id:string)=>read<PublicEvent>(`/events/${encodeURIComponent(id)}`)

