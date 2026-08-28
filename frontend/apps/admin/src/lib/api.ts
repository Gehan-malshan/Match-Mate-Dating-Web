const accountBase = import.meta.env.VITE_ACCOUNT_API_URL ?? 'http://localhost:8081/api/v1'
const eventBase = import.meta.env.VITE_EVENT_API_URL ?? 'http://localhost:8082/api/v1'
const matchmakingBase = import.meta.env.VITE_MATCHMAKING_API_URL ?? 'http://localhost:8083/api/v1'

let accessToken = ''
let refreshRequest: Promise<boolean> | undefined

export type Problem = {
  code: string
  detail: string
  fieldErrors?: Record<string, string>
  status?: number
}

export type AdminAccount = {
  id: string
  email?: string
  roles: string[]
}

export type EventStatus = 'DRAFT' | 'PUBLISHED' | 'REGISTRATION_OPEN' | 'REGISTRATION_CLOSED' | 'CANCELLED'

export type ManagedEvent = {
  eventId: string
  organizerId: string
  name: string
  description: string
  venueName: string
  broadLocation: string
  timeZone: string
  startsAt: string
  endsAt: string
  registrationOpensAt: string
  registrationClosesAt: string
  price: string
  currency: string
  configuredCapacity: number
  capacityPolicyVersion: number
  matchingRulesetVersion: string
  status: EventStatus
  version: number
}

export type EventInput = Omit<ManagedEvent, 'eventId' | 'status' | 'version' | 'capacityPolicyVersion'>

export type Pairing = {
  pairingId?: string
  participantA: string
  participantB: string
  participantACode?: string
  participantBCode?: string
  score: number
  safeReasons: string[]
  source?: 'ALGORITHM' | 'OVERRIDE'
}

export type Candidate = {
  participantA: string
  participantB: string
  eligible: boolean
  rejectionCodes?: string[]
  components?: Record<string, number>
  totalScore?: number
  safeReasons?: string[]
}

export type Unmatched = {
  participantId: string
  participantCode?: string
  reason: string
}

export type MatchingRun = {
  runId: string
  eventId: string
  runVersion: number
  version: number
  status: 'GENERATED' | 'UNDER_REVIEW' | 'LOCKED' | 'PUBLISHED' | 'INVALIDATED'
  rulesetVersion: string
  optimizerVersion: string
  tieBreakPolicy: string
  participantCount: number
  eligiblePairCount: number
  createdBy: string
  createdAt: string
  updatedAt: string
  suggestions?: Pairing[]
  selections?: Pairing[]
  unmatched?: Unmatched[]
  candidates?: Candidate[]
}

async function call<T>(base: string, path: string, init: RequestInit = {}, retry = true): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (init.body) headers.set('Content-Type', 'application/json')
  if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`)
  const response = await fetch(base + path, { ...init, headers, credentials: 'include' })
  if (response.status === 401 && retry && path !== '/auth/refresh') {
    if (await refresh()) return call<T>(base, path, init, false)
  }
  if (!response.ok) throw await response.json() as Problem
  return response.status === 204 ? undefined as T : response.json() as Promise<T>
}

function requireAdmin(account: AdminAccount): AdminAccount {
  if (!account.roles.includes('admin')) {
    accessToken = ''
    throw {
      code: 'ADMIN_ROLE_REQUIRED',
      detail: 'This workspace is restricted to MatchMate administrators.',
      status: 403,
    } satisfies Problem
  }
  return account
}

async function currentAccount() {
  const me = await call<{ account: AdminAccount }>(accountBase, '/users/me')
  return requireAdmin(me.account)
}

export async function login(email: string, password: string) {
  const auth = await call<{ accessToken: string }>(accountBase, '/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
  accessToken = auth.accessToken
  return currentAccount()
}

async function performRefresh() {
  try {
    const auth = await call<{ accessToken: string }>(accountBase, '/auth/refresh', { method: 'POST' }, false)
    accessToken = auth.accessToken
    return true
  } catch {
    accessToken = ''
    return false
  }
}

export function refresh() {
  if (!refreshRequest) {
    refreshRequest = performRefresh().finally(() => {
      refreshRequest = undefined
    })
  }
  return refreshRequest
}

export async function restoreAdminSession() {
  if (!await refresh()) return undefined
  try {
    return await currentAccount()
  } catch {
    return undefined
  }
}

export async function logout() {
  try {
    await call<void>(accountBase, '/auth/logout', { method: 'POST' }, false)
  } finally {
    accessToken = ''
  }
}

export const listManagedEvents = () => call<{ items: ManagedEvent[]; nextCursor?: string; limit: number }>(eventBase, '/organizer/events?limit=100')

export const createEvent = (input: EventInput) => call<ManagedEvent>(eventBase, '/events', {
  method: 'POST',
  body: JSON.stringify(input),
})

export const updateEvent = (eventId: string, input: EventInput, expectedVersion: number) => call<ManagedEvent>(eventBase, `/events/${encodeURIComponent(eventId)}`, {
  method: 'PATCH',
  body: JSON.stringify({ ...input, expectedVersion }),
})

export const transitionEvent = (event: ManagedEvent, action: 'publish' | 'open-registration' | 'close-registration' | 'cancel', reason = '') => call<ManagedEvent>(eventBase, `/events/${encodeURIComponent(event.eventId)}/${action}`, {
  method: 'POST',
  body: JSON.stringify({ expectedVersion: event.version, reason }),
})

function commandKey(prefix: string) {
  return `${prefix}-${crypto.randomUUID()}`
}

export const listMatchingRuns = (eventId: string) => call<{ items: MatchingRun[] }>(matchmakingBase, `/events/${encodeURIComponent(eventId)}/matching-runs`)

export const getMatchingRun = (runId: string) => call<MatchingRun>(matchmakingBase, `/matching-runs/${encodeURIComponent(runId)}`)

export const generateMatchingRun = (eventId: string) => call<MatchingRun>(matchmakingBase, `/events/${encodeURIComponent(eventId)}/matching-runs`, {
  method: 'POST',
  headers: { 'Idempotency-Key': commandKey('generate') },
})

export const reviewMatchingRun = (run: MatchingRun) => call<MatchingRun>(matchmakingBase, `/matching-runs/${encodeURIComponent(run.runId)}/review`, {
  method: 'POST',
  body: JSON.stringify({ expectedVersion: run.version }),
})

export const lockMatchingRun = (run: MatchingRun) => call<MatchingRun>(matchmakingBase, `/matching-runs/${encodeURIComponent(run.runId)}/lock`, {
  method: 'POST',
  headers: { 'Idempotency-Key': commandKey('lock') },
  body: JSON.stringify({ expectedVersion: run.version }),
})

export const publishMatchingRun = (run: MatchingRun) => call<MatchingRun>(matchmakingBase, `/matching-runs/${encodeURIComponent(run.runId)}/publish`, {
  method: 'POST',
  headers: { 'Idempotency-Key': commandKey('publish') },
  body: JSON.stringify({ expectedVersion: run.version }),
})

export const overridePairing = (run: MatchingRun, input: { removeSelectionId: string; participantA: string; participantB: string; reason: string }) => call<MatchingRun>(matchmakingBase, `/matching-runs/${encodeURIComponent(run.runId)}/overrides`, {
  method: 'POST',
  headers: { 'Idempotency-Key': commandKey('override') },
  body: JSON.stringify({ ...input, expectedVersion: run.version }),
})
