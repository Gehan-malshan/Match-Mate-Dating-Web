import { GraphQLClient, type GraphQLProblem } from '@matchmate/graphql-client'

export const graphqlClient = new GraphQLClient(import.meta.env.VITE_GRAPHQL_API_URL ?? 'http://localhost:8080/graphql')
export type ApiProblem = GraphQLProblem
export type Me = { account:{id:string;email:string;status:string;verification:string;roles:string[]};profile:{accountID:string;nickname:string;dateOfBirth:string;broadLocation:string;bio:string;visibility:string;approval:string;interests:string[];version:number};preferences?:{minAge:number;maxAge:number;intentions:string[];interestedIn:string[];languages:string[];dealBreakers:string[]} }
export type CommunityProfile = { profileId:string;nickname:string;ageBand:string;broadLocation:string;bio:string;interests:string[] }
export type CommunityPage = { items:CommunityProfile[];nextCursor?:string }

const meFields = `account { id email status verification roles } profile { accountID nickname dateOfBirth broadLocation bio visibility approval interests version } preferences { minAge maxAge intentions interestedIn languages dealBreakers }`

export async function register(input:{email:string;password:string;nickname:string;dateOfBirth:string;consentVersion:string}) {
  const data=await graphqlClient.execute<{register:{me:Me;verificationToken?:string}}>(`mutation Register($input:RegisterInput!){register(input:$input){me{${meFields}} verificationToken}}`,{input},false)
  return data.register
}
export async function verifyEmail(token:string){await graphqlClient.execute(`mutation VerifyEmail($token:String!){verifyEmail(token:$token){success}}`,{token},false)}
export async function resendVerification(email:string){await graphqlClient.execute(`mutation ResendVerification($email:String!){resendVerification(email:$email){success}}`,{email},false)}
export async function requestPasswordReset(email:string){await graphqlClient.execute(`mutation RequestPasswordReset($email:String!){requestPasswordReset(email:$email){success}}`,{email},false)}
export async function resetPassword(token:string,password:string){await graphqlClient.execute(`mutation ResetPassword($token:String!,$password:String!){resetPassword(token:$token,password:$password){success}}`,{token,password},false)}
export async function deleteMyAccount(password:string){await graphqlClient.execute(`mutation DeleteMyAccount($password:String!){deleteMyAccount(password:$password){success}}`,{password});graphqlClient.clearToken()}
export async function login(email:string,password:string,totpCode=''){const data=await graphqlClient.execute<{login:{accessToken:string;me:Me}}>(`mutation Login($email:String!,$password:String!,$totpCode:String){login(email:$email,password:$password,totpCode:$totpCode){accessToken me{${meFields}}}}`,{email,password,totpCode},false);graphqlClient.setToken(data.login.accessToken);return data.login}
export const refresh=()=>graphqlClient.refresh()
export async function logout(){try{await graphqlClient.execute(`mutation Logout{logout{success}}`,{},false)}finally{graphqlClient.clearToken()}}
export async function getMe(){const data=await graphqlClient.execute<{me:Me}>(`query Me{me{${meFields}}}`);return data.me}
export async function updateProfile(profile:Record<string,unknown>){const data=await graphqlClient.execute<{updateProfile:Me['profile']}>(`mutation UpdateProfile($input:ProfileInput!){updateProfile(input:$input){accountID nickname dateOfBirth broadLocation bio visibility approval interests version}}`,{input:profile});return data.updateProfile}
export async function updatePreferences(preferences:Record<string,unknown>){const data=await graphqlClient.execute<{updatePreferences:NonNullable<Me['preferences']>}>(`mutation UpdatePreferences($input:PreferencesInput!){updatePreferences(input:$input){minAge maxAge intentions interestedIn languages dealBreakers}}`,{input:preferences});return data.updatePreferences}
export async function listCommunityProfiles(cursor=''){const data=await graphqlClient.execute<{communityProfiles:CommunityPage}>(`query Community($cursor:String){communityProfiles(limit:8,cursor:$cursor){items{profileId nickname ageBand broadLocation bio interests} nextCursor}}`,{cursor:cursor||null});return data.communityProfiles}
export async function getCommunityProfile(profileId:string){const data=await graphqlClient.execute<{communityProfile:CommunityProfile}>(`query CommunityProfile($profileId:ID!){communityProfile(profileId:$profileId){profileId nickname ageBand broadLocation bio interests}}`,{profileId});return data.communityProfile}
export async function blockMember(accountId:string){await graphqlClient.execute(`mutation Block($accountId:ID!){blockMember(accountId:$accountId){success}}`,{accountId})}
