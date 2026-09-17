export type GraphQLProblem = {
  code: string
  detail: string
  status?: number
  fields?: Record<string, string>
  fieldErrors?: Record<string, string>
}

type GraphQLErrorResponse = {
  message: string
  extensions?: { code?: string; httpStatus?: number; fields?: Record<string, string>; fieldErrors?: Record<string, string> }
}

type Envelope<T> = { data?: T; errors?: GraphQLErrorResponse[] }
const refreshMutation = `mutation RefreshSession { refreshSession { accessToken } }`

export class GraphQLClient {
  private accessToken = ''
  private refreshRequest?: Promise<boolean>
  constructor(private readonly endpoint: string) {}

  async execute<T>(query: string, variables: Record<string, unknown> = {}, retry = true): Promise<T> {
    try { return await this.raw<T>(query, variables) }
    catch (problem) {
      if (retry && (problem as GraphQLProblem).status === 401 && await this.refresh()) return this.raw<T>(query, variables)
      throw problem
    }
  }

  setToken(value: string) { this.accessToken = value }
  clearToken() { this.accessToken = '' }
  refresh() {
    if (!this.refreshRequest) {
      this.refreshRequest = this.raw<{ refreshSession: { accessToken: string } }>(refreshMutation)
        .then((result) => { this.accessToken = result.refreshSession.accessToken; return true })
        .catch(() => { this.accessToken = ''; return false })
        .finally(() => { this.refreshRequest = undefined })
    }
    return this.refreshRequest
  }

  private async raw<T>(query: string, variables: Record<string, unknown> = {}): Promise<T> {
    const headers = new Headers({ 'Content-Type': 'application/json', Accept: 'application/json' })
    if (this.accessToken) headers.set('Authorization', `Bearer ${this.accessToken}`)
    const response = await fetch(this.endpoint, { method: 'POST', headers, credentials: 'include', body: JSON.stringify({ query, variables }) })
    const envelope = await response.json() as Envelope<T>
    if (!response.ok || envelope.errors?.length) {
      const first = envelope.errors?.[0]
      throw { code: first?.extensions?.code ?? 'GRAPHQL_REQUEST_FAILED', detail: first?.message ?? `GraphQL request failed with status ${response.status}.`, status: first?.extensions?.httpStatus ?? response.status, fields: first?.extensions?.fields, fieldErrors: first?.extensions?.fieldErrors } satisfies GraphQLProblem
    }
    if (!envelope.data) throw { code: 'EMPTY_GRAPHQL_RESPONSE', detail: 'The GraphQL response did not include data.' } satisfies GraphQLProblem
    return envelope.data
  }
}
