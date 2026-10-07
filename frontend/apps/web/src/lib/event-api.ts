import {graphqlClient} from './account-api'

export type EventStatus='PUBLISHED'|'REGISTRATION_OPEN'|'REGISTRATION_CLOSED'
export type PublicEvent={eventId:string;name:string;description:string;broadLocation:string;timeZone:string;startsAt:string;endsAt:string;registrationOpensAt:string;registrationClosesAt:string;price:string;currency:string;paymentOptions:'ONLINE'|'AT_VENUE'|'BOTH';imageVersion:number;configuredCapacity:number;matchingRulesetVersion:string;status:EventStatus;version:number}
export type EventPage={items:PublicEvent[];nextCursor?:string;limit:number}
const fields='eventId name description broadLocation timeZone startsAt endsAt registrationOpensAt registrationClosesAt price currency paymentOptions imageVersion configuredCapacity matchingRulesetVersion status version'
export function eventImageUrl(event:Pick<PublicEvent,'eventId'|'imageVersion'>):string|undefined {
  if (!event.imageVersion) return undefined
  const endpoint = import.meta.env.VITE_GRAPHQL_API_URL ?? 'http://localhost:8080/graphql'
  const origin = new URL(endpoint, globalThis.location?.origin ?? 'http://localhost:8080').origin
  return new URL(`/media/events/${encodeURIComponent(event.eventId)}?v=${event.imageVersion}`, origin).toString()
}
export async function listEvents(cursor=''){const data=await graphqlClient.execute<{events:EventPage}>(`query Events($cursor:String){events(limit:12,cursor:$cursor){items{${fields}} nextCursor limit}}`,{cursor:cursor||null},false);return data.events}
export async function getEvent(eventId:string){const data=await graphqlClient.execute<{event:PublicEvent}>(`query Event($eventId:ID!){event(eventId:$eventId){${fields}}}`,{eventId},false);return data.event}
