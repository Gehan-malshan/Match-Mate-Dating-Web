import {Outlet,createRootRoute,createRoute,createRouter,redirect} from '@tanstack/react-router'
import {lazy,Suspense} from 'react'
import {getMe} from './lib/account-api'
import {LandingPage} from './routes/LandingPage'
import {LoginPage} from './routes/LoginPage'
import {ProfilePage} from './routes/ProfilePage'
import {RegisterPage} from './routes/RegisterPage'
import {EventsPage} from './routes/EventsPage'
import {EventDetailPage} from './routes/EventDetailPage'
import {CommunityPage} from './routes/CommunityPage'
import {CommunityProfilePage} from './routes/CommunityProfilePage'
import {BookingsPage} from './routes/BookingsPage'
import {NotificationsPage} from './routes/NotificationsPage'
const AdminPage=lazy(()=>import('./routes/AdminPage').then((module)=>({default:module.AdminPage})))
const AdminRoute=()=> <Suspense fallback={<main className="admin-loading"><p>Opening the protected workspace…</p></main>}><AdminPage/></Suspense>

const rootRoute=createRootRoute({component:Outlet})
const indexRoute=createRoute({getParentRoute:()=>rootRoute,path:'/',component:LandingPage})
const registerRoute=createRoute({getParentRoute:()=>rootRoute,path:'/register',component:RegisterPage})
const loginRoute=createRoute({getParentRoute:()=>rootRoute,path:'/login',component:LoginPage})
const eventsRoute=createRoute({getParentRoute:()=>rootRoute,path:'/events',component:EventsPage})
const eventDetailRoute=createRoute({getParentRoute:()=>rootRoute,path:'/events/$eventId',component:EventDetailPage})

const memberGroup=createRoute({
  getParentRoute:()=>rootRoute,
  id:'member-access',
  beforeLoad:async()=>{try{return{me:await getMe()}}catch{throw redirect({to:'/login'})}},
  component:Outlet,
})
const profileRoute=createRoute({getParentRoute:()=>memberGroup,path:'/app/profile',component:ProfilePage})
const bookingsRoute=createRoute({getParentRoute:()=>memberGroup,path:'/app/bookings',component:BookingsPage})
const notificationsRoute=createRoute({getParentRoute:()=>memberGroup,path:'/app/notifications',component:NotificationsPage})
const communityRoute=createRoute({getParentRoute:()=>memberGroup,path:'/community',component:CommunityPage})
const communityProfileRoute=createRoute({getParentRoute:()=>memberGroup,path:'/community/$profileId',component:CommunityProfilePage})

const adminRoute=createRoute({
  getParentRoute:()=>rootRoute,
  path:'/admin',
  beforeLoad:async()=>{try{const me=await getMe();if(!me.account.roles.includes('admin'))throw new Error('forbidden');return{me}}catch{throw redirect({to:'/login'})}},
  component:AdminRoute,
})

const routeTree=rootRoute.addChildren([indexRoute,registerRoute,loginRoute,eventsRoute,eventDetailRoute,adminRoute,memberGroup.addChildren([profileRoute,bookingsRoute,notificationsRoute,communityRoute,communityProfileRoute])])
export const router=createRouter({routeTree})
declare module '@tanstack/react-router'{interface Register{router:typeof router}}
