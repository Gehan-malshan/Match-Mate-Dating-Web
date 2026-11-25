import {describe,expect,it} from 'vitest'
import {findNewUnreadNotifications} from './NotificationCenter'
import type {NotificationItem} from '../lib/notification-api'

const item=(id:string,readAt?:string):NotificationItem=>({notificationId:id,sourceEventType:'BookingPending',category:'BOOKING',title:'Seat held',message:'Complete payment.',actionPath:'/app/bookings',createdAt:'2026-08-30T10:00:00Z',readAt})

describe('notification popup selection',()=>{
  it('shows only new unread records and never repeats a seen notification',()=>{
    const fresh=findNewUnreadNotifications(new Set(['known']),[item('known'),item('new'),item('read','2026-08-30T10:01:00Z')])
    expect(fresh.map((notification)=>notification.notificationId)).toEqual(['new'])
  })
})
