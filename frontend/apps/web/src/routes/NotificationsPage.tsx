import {useInfiniteQuery,useMutation,useQueryClient} from '@tanstack/react-query'
import {Link} from '@tanstack/react-router'
import {MemberNavigation} from '../components/MemberNavigation'
import {listNotifications,markAllNotificationsRead,markNotificationRead,type NotificationItem} from '../lib/notification-api'
import './NotificationsPage.css'

function destination(item:NotificationItem):'/app/profile'|'/app/bookings'{
  return item.actionPath==='/app/bookings'?'/app/bookings':'/app/profile'
}

export function NotificationsPage(){
  const client=useQueryClient()
  const query=useInfiniteQuery({
    queryKey:['notifications','history'],
    initialPageParam:'',
    queryFn:({pageParam})=>listNotifications(20,pageParam),
    getNextPageParam:(last)=>last.nextCursor,
    retry:false,
  })
  const refresh=()=>client.invalidateQueries({queryKey:['notifications']})
  const mark=useMutation({mutationFn:markNotificationRead,onSuccess:refresh})
  const markAll=useMutation({mutationFn:markAllNotificationsRead,onSuccess:refresh})
  const items=query.data?.pages.flatMap((page)=>page.items)??[]
  const unread=query.data?.pages[0]?.unreadCount??0

  return <main className="notifications-page">
    <MemberNavigation active="notifications"/>
    <section className="notifications-shell">
      <header><div><p>Private member updates</p><h1>Your notifications</h1><span>Account and booking changes appear here without exposing contact details or private matching answers.</span></div>{unread>0&&<button type="button" disabled={markAll.isPending} onClick={()=>markAll.mutate()}>Mark all as read</button>}</header>
      {query.isPending&&<p className="notifications-status" role="status">Loading your notifications…</p>}
      {query.isError&&<div className="notifications-empty"><h2>Sign in to view your notifications</h2><p>This history belongs only to the authenticated member.</p><Link className="button" to="/login">Log in</Link></div>}
      {!query.isPending&&!query.isError&&items.length===0&&<div className="notifications-empty"><h2>You are all caught up</h2><p>New account and booking updates will appear here.</p><Link className="button" to="/events">Explore events</Link></div>}
      <div className="notifications-list">{items.map((item)=><article className={item.readAt?'':'is-unread'} key={item.notificationId}>
        <span className="notifications-marker" aria-hidden="true"/>
        <div><div className="notifications-meta"><span>{item.category==='BOOKING'?'Booking update':'Account update'}</span><time dateTime={item.createdAt}>{new Date(item.createdAt).toLocaleString('en-LK')}</time></div><h2>{item.title}</h2><p>{item.message}</p><div className="notifications-actions"><Link to={destination(item)} onClick={()=>{if(!item.readAt)mark.mutate(item.notificationId)}}>Open related page <span aria-hidden="true">→</span></Link>{!item.readAt&&<button type="button" disabled={mark.isPending} onClick={()=>mark.mutate(item.notificationId)}>Mark as read</button>}</div></div>
      </article>)}</div>
      {query.hasNextPage&&<button className="notifications-load" type="button" disabled={query.isFetchingNextPage} onClick={()=>query.fetchNextPage()}>{query.isFetchingNextPage?'Loading…':'Load older notifications'}</button>}
    </section>
  </main>
}
