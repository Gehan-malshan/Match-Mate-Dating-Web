import {graphqlClient} from './account-api'

export type NotificationItem={notificationId:string;sourceEventType:string;category:'ACCOUNT'|'BOOKING'|string;title:string;message:string;actionPath:'/app/profile'|'/app/bookings'|string;readAt?:string;createdAt:string}
export type NotificationPage={items:NotificationItem[];nextCursor?:string;unreadCount:number}
export type NotificationPreference={channel:'EMAIL'|string;category:'ACCOUNT'|'BOOKING'|string;allowed:boolean;updatedAt?:string}
const fields='notificationId sourceEventType category title message actionPath readAt createdAt'
export async function listNotifications(limit=20,cursor=''){const data=await graphqlClient.execute<{notifications:NotificationPage}>(`query Notifications($limit:Int!,$cursor:String){notifications(limit:$limit,cursor:$cursor){items{${fields}} nextCursor unreadCount}}`,{limit,cursor:cursor||null});return data.notifications}
export async function getUnreadNotificationCount(){const data=await graphqlClient.execute<{unreadNotificationCount:{unreadCount:number}}>(`query UnreadCount{unreadNotificationCount{unreadCount}}`);return data.unreadNotificationCount}
export async function markNotificationRead(notificationId:string){await graphqlClient.execute(`mutation MarkRead($notificationId:ID!){markNotificationRead(notificationId:$notificationId){success}}`,{notificationId})}
export async function markAllNotificationsRead(){const data=await graphqlClient.execute<{markAllNotificationsRead:{updatedCount:number}}>(`mutation MarkAllRead{markAllNotificationsRead{updatedCount}}`);return data.markAllNotificationsRead}
export async function listNotificationPreferences(){const data=await graphqlClient.execute<{notificationPreferences:NotificationPreference[]}>(`query NotificationPreferences{notificationPreferences{channel category allowed updatedAt}}`);return data.notificationPreferences}
export async function updateNotificationPreference(category:string,emailEnabled:boolean){const data=await graphqlClient.execute<{updateNotificationPreference:NotificationPreference}>(`mutation UpdateNotificationPreference($input:NotificationPreferenceInput!){updateNotificationPreference(input:$input){channel category allowed updatedAt}}`,{input:{category,emailEnabled}});return data.updateNotificationPreference}
