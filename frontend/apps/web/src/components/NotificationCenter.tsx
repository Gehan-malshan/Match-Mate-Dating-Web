import {useEffect,useRef,useState} from 'react'
import {useMutation,useQuery,useQueryClient} from '@tanstack/react-query'
import {Link} from '@tanstack/react-router'
import {listNotifications,markAllNotificationsRead,markNotificationRead,type NotificationItem} from '../lib/notification-api'
import './NotificationCenter.css'

export function findNewUnreadNotifications(seen:Set<string>,items:NotificationItem[]){
  return items.filter((item)=>!item.readAt&&!seen.has(item.notificationId))
}

function destination(item:NotificationItem):'/app/profile'|'/app/bookings'{
  return item.actionPath==='/app/bookings'?'/app/bookings':'/app/profile'
}

export function NotificationCenter(){
  const [open,setOpen]=useState(false)
  const [toasts,setToasts]=useState<NotificationItem[]>([])
  const initialized=useRef(false)
  const seen=useRef(new Set<string>())
  const client=useQueryClient()
  const query=useQuery({
    queryKey:['notifications','preview'],
    queryFn:()=>listNotifications(6),
    refetchInterval:10000,
    retry:false,
  })
  const refresh=()=>client.invalidateQueries({queryKey:['notifications']})
  const mark=useMutation({mutationFn:markNotificationRead,onSuccess:refresh})
  const markAll=useMutation({mutationFn:markAllNotificationsRead,onSuccess:refresh})

  useEffect(()=>{
    if(!query.data)return
    if(!initialized.current){
      query.data.items.forEach((item)=>seen.current.add(item.notificationId))
      initialized.current=true
      return
    }
    const fresh=findNewUnreadNotifications(seen.current,query.data.items)
    query.data.items.forEach((item)=>seen.current.add(item.notificationId))
    if(fresh.length)setToasts((current)=>[...fresh,...current].slice(0,3))
  },[query.data])

  useEffect(()=>{
    if(!toasts.length)return
    const timer=window.setTimeout(()=>setToasts((current)=>current.slice(0,-1)),6000)
    return()=>window.clearTimeout(timer)
  },[toasts])

  if(query.isError)return null
  const unread=query.data?.unreadCount??0
  return <div className="notification-center">
    <button className="notification-bell" type="button" aria-label={unread?'Open notifications, '+unread+' unread':'Open notifications'} aria-expanded={open} onClick={()=>setOpen(!open)}>
      <svg aria-hidden="true" viewBox="0 0 24 24"><path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4"/></svg>
      {unread>0&&<span>{unread>99?'99+':unread}</span>}
    </button>
    {open&&<section className="notification-popover" aria-label="Recent notifications">
      <header><div><small>Your updates</small><h2>Notifications</h2></div>{unread>0&&<button type="button" disabled={markAll.isPending} onClick={()=>markAll.mutate()}>Mark all read</button>}</header>
      {query.isPending&&<p className="notification-state" role="status">Loading your updates…</p>}
      {query.data?.items.length===0&&<p className="notification-state">You are all caught up.</p>}
      <div className="notification-preview-list">{query.data?.items.map((item)=><Link key={item.notificationId} className={item.readAt?'':'is-unread'} to={destination(item)} onClick={()=>{setOpen(false);if(!item.readAt)mark.mutate(item.notificationId)}}>
        <span className="notification-dot" aria-hidden="true"/>
        <span><strong>{item.title}</strong><small>{item.message}</small><time dateTime={item.createdAt}>{new Date(item.createdAt).toLocaleString('en-LK')}</time></span>
      </Link>)}</div>
      <Link className="notification-view-all" to="/app/notifications" onClick={()=>setOpen(false)}>View all notifications <span aria-hidden="true">→</span></Link>
    </section>}
    <div className="notification-toast-region" aria-live="polite" aria-label="New notifications">{toasts.map((item)=><article className="notification-toast" key={item.notificationId}>
      <div><small>New MatchMate update</small><strong>{item.title}</strong><p>{item.message}</p></div>
      <div><Link to={destination(item)} onClick={()=>{mark.mutate(item.notificationId);setToasts((current)=>current.filter((toast)=>toast.notificationId!==item.notificationId))}}>View</Link><button type="button" aria-label={'Dismiss '+item.title} onClick={()=>setToasts((current)=>current.filter((toast)=>toast.notificationId!==item.notificationId))}>×</button></div>
    </article>)}</div>
  </div>
}
