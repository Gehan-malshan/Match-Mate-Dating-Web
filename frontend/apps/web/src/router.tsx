import {
  Outlet,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router'
import { LandingPage } from './routes/LandingPage'
import { LoginPage } from './routes/LoginPage'
import { ProfilePage } from './routes/ProfilePage'
import { RegisterPage } from './routes/RegisterPage'

const rootRoute = createRootRoute({ component: Outlet })

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: LandingPage,
})
const registerRoute=createRoute({getParentRoute:()=>rootRoute,path:'/register',component:RegisterPage})
const loginRoute=createRoute({getParentRoute:()=>rootRoute,path:'/login',component:LoginPage})
const profileRoute=createRoute({getParentRoute:()=>rootRoute,path:'/app/profile',component:ProfilePage})

const routeTree = rootRoute.addChildren([indexRoute,registerRoute,loginRoute,profileRoute])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
