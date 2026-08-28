const baseUrl = import.meta.env.VITE_ACCOUNT_API_URL ?? 'http://localhost:8081/api/v1'
let accessToken = ''

export type ApiProblem = { code: string; detail: string; fields?: Record<string,string> }
export type Me = { account:{id:string;email:string;status:string;verification:string;roles:string[]};profile:{accountID:string;nickname:string;dateOfBirth:string;broadLocation:string;bio:string;visibility:string;approval:string;interests:string[];version:number};preferences?:{minAge:number;maxAge:number;intentions:string[];interestedIn:string[];languages:string[];dealBreakers:string[]} }

async function request<T>(path:string,init:RequestInit={},retry=true):Promise<T>{
  const headers = new Headers(init.headers); headers.set('Content-Type','application/json'); if(accessToken) headers.set('Authorization',`Bearer ${accessToken}`)
  const response=await fetch(`${baseUrl}${path}`,{...init,headers,credentials:'include'})
  if(response.status===401&&retry&&path!=='/auth/refresh'){const refreshed=await refresh();if(refreshed)return request<T>(path,init,false)}
  if(!response.ok){throw await response.json() as ApiProblem}
  return response.status===204 ? undefined as T : response.json() as Promise<T>
}
export async function register(input:{email:string;password:string;nickname:string;dateOfBirth:string;consentVersion:string}){return request<{me:Me;verificationToken?:string}>('/auth/register',{method:'POST',body:JSON.stringify(input)})}
export async function verifyEmail(token:string){return request('/auth/verify-email',{method:'POST',body:JSON.stringify({token})})}
export async function login(email:string,password:string){const result=await request<{accessToken:string}>('/auth/login',{method:'POST',body:JSON.stringify({email,password})});accessToken=result.accessToken;return result}
export async function refresh(){try{const result=await request<{accessToken:string}>('/auth/refresh',{method:'POST'},false);accessToken=result.accessToken;return true}catch{accessToken='';return false}}
export async function logout(){await request('/auth/logout',{method:'POST'},false);accessToken=''}
export const getMe=()=>request<Me>('/users/me')
export const updateProfile=(profile:Record<string,unknown>)=>request<Me['profile']>('/users/me/profile',{method:'PATCH',body:JSON.stringify(profile)})
export const updatePreferences=(preferences:Record<string,unknown>)=>request<Me['preferences']>('/users/me/matching-preferences',{method:'PUT',body:JSON.stringify(preferences)})
