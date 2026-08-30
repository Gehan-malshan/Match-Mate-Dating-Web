import {authenticatedRequest} from './account-api'

const notificationBase=import.meta.env.VITE_NOTIFICATION_API_URL??'http://localhost:8086/api/v1'

export type NotificationItem={
  notificationId:string
  sourceEventType:string
  category:'ACCOUNT'|'BOOKING'|string
  title:string
  message:string
  actionPath:'/app/profile'|'/app/bookings'|string
  readAt?:string
  createdAt:string
}

export type NotificationPage={
  items:NotificationItem[]
  nextCursor?:string
  unreadCount:number
}

export const listNotifications=(limit=20,cursor='')=>authenticatedRequest<NotificationPage>(
  notificationBase,
  `/notifications?limit=${limit}${cursor?`&cursor=${encodeURIComponent(cursor)}`:''}`,
)

export const getUnreadNotificationCount=()=>authenticatedRequest<{unreadCount:number}>(
  notificationBase,
  '/notifications/unread-count',
)

export const markNotificationRead=(notificationId:string)=>authenticatedRequest<void>(
  notificationBase,
  `/notifications/${encodeURIComponent(notificationId)}/read`,
  {method:'PATCH'},
)

export const markAllNotificationsRead=()=>authenticatedRequest<{updatedCount:number}>(
  notificationBase,
  '/notifications/read-all',
  {method:'POST'},
)
