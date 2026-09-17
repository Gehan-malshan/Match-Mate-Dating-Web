import {graphqlClient} from './account-api'

export type EventStatus='PUBLISHED'|'REGISTRATION_OPEN'|'REGISTRATION_CLOSED'
export type PublicEvent={eventId:string;name:string;description:string;broadLocation:string;timeZone:string;startsAt:string;endsAt:string;registrationOpensAt:string;registrationClosesAt:string;price:string;currency:string;configuredCapacity:number;matchingRulesetVersion:string;status:EventStatus;version:number}
export type EventPage={items:PublicEvent[];nextCursor?:string;limit:number}
const fields='eventId name description broadLocation timeZone startsAt endsAt registrationOpensAt registrationClosesAt price currency configuredCapacity matchingRulesetVersion status version'
export async function listEvents(cursor=''){const data=await graphqlClient.execute<{events:EventPage}>(`query Events($cursor:String){events(limit:12,cursor:$cursor){items{${fields}} nextCursor limit}}`,{cursor:cursor||null},false);return data.events}
export async function getEvent(eventId:string){const data=await graphqlClient.execute<{event:PublicEvent}>(`query Event($eventId:ID!){event(eventId:$eventId){${fields}}}`,{eventId},false);return data.event}
