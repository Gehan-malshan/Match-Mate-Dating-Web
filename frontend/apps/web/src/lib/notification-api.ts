import {graphqlClient} from './account-api'

export type NotificationItem={notificationId:string;sourceEventType:string;category:'ACCOUNT'|'BOOKING'|string;title:string;message:string;actionPath:'/app/profile'|'/app/bookings'|string;readAt?:string;createdAt:string}
export type NotificationPage={items:NotificationItem[];nextCursor?:string;unreadCount:number}
const fields='notificationId sourceEventType category title message actionPath readAt createdAt'
export async function listNotifications(limit=20,cursor=''){const data=await graphqlClient.execute<{notifications:NotificationPage}>(`query Notifications($limit:Int!,$cursor:String){notifications(limit:$limit,cursor:$cursor){items{${fields}} nextCursor unreadCount}}`,{limit,cursor:cursor||null});return data.notifications}
export async function getUnreadNotificationCount(){const data=await graphqlClient.execute<{unreadNotificationCount:{unreadCount:number}}>(`query UnreadCount{unreadNotificationCount{unreadCount}}`);return data.unreadNotificationCount}
export async function markNotificationRead(notificationId:string){await graphqlClient.execute(`mutation MarkRead($notificationId:ID!){markNotificationRead(notificationId:$notificationId){success}}`,{notificationId})}
export async function markAllNotificationsRead(){const data=await graphqlClient.execute<{markAllNotificationsRead:{updatedCount:number}}>(`mutation MarkAllRead{markAllNotificationsRead{updatedCount}}`);return data.markAllNotificationsRead}
