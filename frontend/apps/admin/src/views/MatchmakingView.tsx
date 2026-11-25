import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  generateMatchingRun,
  getMatchingRun,
  listMatchingRuns,
  lockMatchingRun,
  ManagedEvent,
  MatchingRun,
  overridePairing,
  Pairing,
  Problem,
  publishMatchingRun,
  reviewMatchingRun,
} from '../lib/api'

const fixtureEventId = '11111111-1111-4111-8111-000000000001'

export function MatchmakingView({ events, loadingEvents }: { events: ManagedEvent[]; loadingEvents: boolean }) {
  const queryClient = useQueryClient()
  const [eventId, setEventId] = useState('')
  const [selectedRunId, setSelectedRunId] = useState('')
  const [message, setMessage] = useState('')
  const [adjusting, setAdjusting] = useState<Pairing>()

  useEffect(() => {
    if (!eventId && events.length) setEventId(events.find((event) => event.eventId === fixtureEventId)?.eventId ?? events[0].eventId)
  }, [eventId, events])

  const runsQuery = useQuery({
    queryKey: ['matching-runs', eventId],
    queryFn: () => listMatchingRuns(eventId),
    enabled: Boolean(eventId),
    retry: false,
  })

  useEffect(() => {
    const runs = runsQuery.data?.items ?? []
    if (runs.length && !runs.some((run) => run.runId === selectedRunId)) setSelectedRunId(runs[0].runId)
    if (!runs.length) setSelectedRunId('')
  }, [runsQuery.data, selectedRunId])

  const runQuery = useQuery({
    queryKey: ['matching-run', selectedRunId],
    queryFn: () => getMatchingRun(selectedRunId),
    enabled: Boolean(selectedRunId),
  })

  const generateMutation = useMutation({
    mutationFn: () => generateMatchingRun(eventId),
    onSuccess: async (run) => {
      setSelectedRunId(run.runId)
      setMessage(`Run ${run.runVersion} generated from immutable fixture snapshots.`)
      await queryClient.invalidateQueries({ queryKey: ['matching-runs', eventId] })
      queryClient.setQueryData(['matching-run', run.runId], run)
    },
    onError: (problem) => setMessage((problem as unknown as Problem).detail ?? 'Matching generation failed.'),
  })

  const lifecycleMutation = useMutation({
    mutationFn: ({ run, action }: { run: MatchingRun; action: 'review' | 'lock' | 'publish' }) => action === 'review' ? reviewMatchingRun(run) : action === 'lock' ? lockMatchingRun(run) : publishMatchingRun(run),
    onSuccess: async (run) => {
      setMessage(`Run moved to ${run.status.replaceAll('_', ' ').toLowerCase()}.`)
      queryClient.setQueryData(['matching-run', run.runId], run)
      await queryClient.invalidateQueries({ queryKey: ['matching-runs', eventId] })
    },
    onError: (problem) => setMessage((problem as unknown as Problem).detail ?? 'The run could not be updated.'),
  })

  const selectedEvent = events.find((event) => event.eventId === eventId)
  const run = runQuery.data
  const runsProblem = runsQuery.error as unknown as Problem | null

  return (
    <section className="admin-view matchmaking-view">
      <header className="view-heading">
        <div><p className="eyebrow">Rule-based matchmaking</p><h1>The compatibility <em>forge.</em></h1><p>Generate reproducible pair proposals, inspect generalized reasons, review changes, then lock and publish.</p></div>
        <span className="no-ml-badge"><i aria-hidden="true" /> No machine learning</span>
      </header>

      <div className="matching-toolbar panel">
        <label>Event cohort<select value={eventId} disabled={loadingEvents} onChange={(event) => { setEventId(event.target.value); setSelectedRunId(''); setMessage('') }}><option value="">Select an event</option>{events.map((event) => <option key={event.eventId} value={event.eventId}>{event.name} · {event.status.replaceAll('_', ' ')}</option>)}</select></label>
        <div className="matching-event-summary"><span>{selectedEvent?.broadLocation ?? 'No event selected'}</span><strong>{selectedEvent ? formatDate(selectedEvent.startsAt, selectedEvent.timeZone) : 'Choose a cohort to continue'}</strong></div>
        <button className="primary-action" type="button" disabled={!eventId || generateMutation.isPending || Boolean(runsProblem)} onClick={() => generateMutation.mutate()}>{generateMutation.isPending ? 'Generating…' : 'Generate new run'} <span aria-hidden="true">✦</span></button>
      </div>

      {runsProblem && <section className="matching-not-configured" role="alert"><span aria-hidden="true">◇</span><div><h2>Matching inputs are not ready</h2><p>{runsProblem.detail}</p><small>In this prototype, deterministic participant fixtures are available for “Midnight Rooftop Social.” New events will become eligible after Booking and Moderation projections are integrated.</small></div></section>}

      {!runsProblem && eventId && <div className="matching-layout">
        <aside className="panel run-history">
          <div className="panel-heading"><div><p className="eyebrow">Version history</p><h2>Matching runs</h2></div><span>{runsQuery.data?.items.length ?? 0}</span></div>
          {runsQuery.isPending && <p className="empty-state">Loading run history…</p>}
          {!runsQuery.isPending && !runsQuery.data?.items.length && <p className="empty-state">No run exists for this event yet.</p>}
          <div className="run-list">{runsQuery.data?.items.map((item) => <button type="button" className={item.runId === selectedRunId ? 'is-active' : ''} key={item.runId} onClick={() => setSelectedRunId(item.runId)}><span>Run {item.runVersion}<small>{new Date(item.createdAt).toLocaleString('en-LK')}</small></span><i className={`status-badge status-${item.status.toLowerCase()}`}>{item.status.replaceAll('_', ' ')}</i></button>)}</div>
        </aside>

        <div className="matching-main">
          {!selectedRunId && <section className="panel matching-empty"><span aria-hidden="true">◇</span><h2>Ready for a deterministic proposal</h2><p>Generation snapshots approved fixture inputs and stores every candidate decision for audit and replay.</p></section>}
          {selectedRunId && runQuery.isPending && <section className="panel matching-empty"><div className="inline-loader" aria-hidden="true" /><p>Opening the selected run…</p></section>}
          {run && <RunWorkspace run={run} busy={lifecycleMutation.isPending} onLifecycle={(action) => lifecycleMutation.mutate({ run, action })} onAdjust={setAdjusting} />}
        </div>
      </div>}

      {adjusting && run && <OverrideEditor run={run} pairing={adjusting} onClose={() => setAdjusting(undefined)} onSaved={async (updated) => { setAdjusting(undefined); queryClient.setQueryData(['matching-run', updated.runId], updated); await queryClient.invalidateQueries({ queryKey: ['matching-runs', eventId] }); setMessage('The replacement pairing was recorded with an audit reason.') }} />}
      {message && <p className="operation-message matching-message" role="status">{message}</p>}
    </section>
  )
}

function RunWorkspace({ run, busy, onLifecycle, onAdjust }: { run: MatchingRun; busy: boolean; onLifecycle: (action: 'review' | 'lock' | 'publish') => void; onAdjust: (pairing: Pairing) => void }) {
  const pairings = run.selections?.length ? run.selections : run.suggestions ?? []
  const rejectionSummary = useMemo(() => {
    const result = new Map<string, number>()
    run.candidates?.forEach((candidate) => candidate.rejectionCodes?.forEach((code) => result.set(code, (result.get(code) ?? 0) + 1)))
    return [...result.entries()].sort((a, b) => b[1] - a[1])
  }, [run.candidates])

  return <>
    <section className="panel run-overview">
      <div className="run-title"><div><p className="eyebrow">Run {run.runVersion} · ruleset {run.rulesetVersion}</p><h2>{run.status === 'PUBLISHED' ? 'Published connections' : 'Proposed connections'}</h2></div><span className={`status-badge status-${run.status.toLowerCase()}`}>{run.status.replaceAll('_', ' ')}</span></div>
      <div className="run-metrics"><div><strong>{run.participantCount}</strong><span>Participants</span></div><div><strong>{pairings.length}</strong><span>Selected pairs</span></div><div><strong>{run.unmatched?.length ?? 0}</strong><span>Unmatched</span></div><div><strong>{run.eligiblePairCount}</strong><span>Eligible edges</span></div></div>
      <div className="run-actions"><p><strong>Next controlled step</strong><span>{nextStep(run.status)}</span></p>{run.status === 'GENERATED' && <button className="primary-action" disabled={busy} onClick={() => onLifecycle('review')}>Start administrator review</button>}{run.status === 'UNDER_REVIEW' && <button className="primary-action" disabled={busy} onClick={() => onLifecycle('lock')}>Revalidate and lock pairings</button>}{run.status === 'LOCKED' && <button className="primary-action" disabled={busy} onClick={() => onLifecycle('publish')}>Publish member-safe matches</button>}{run.status === 'PUBLISHED' && <span className="published-seal">✓ Publication complete</span>}</div>
    </section>

    <section className="connections-section">
      <div className="section-heading"><div><p className="eyebrow">Explainable output</p><h2>Matched connections</h2></div><p>Only participant codes and generalized reasons are displayed.</p></div>
      <div className="pairing-grid">{pairings.map((pairing) => <PairingCard key={pairing.pairingId ?? `${pairing.participantA}-${pairing.participantB}`} pairing={pairing} canAdjust={run.status === 'UNDER_REVIEW'} onAdjust={() => onAdjust(pairing)} />)}</div>
    </section>

    <div className="matching-audit-grid">
      <section className="panel"><div className="panel-heading"><div><p className="eyebrow">Unmatched review</p><h2>Participant outcomes</h2></div></div>{!run.unmatched?.length && <p className="empty-state">Every eligible participant has a proposed connection.</p>}<ul className="outcome-list">{run.unmatched?.map((item) => <li key={item.participantId}><span>{item.participantCode ?? shortId(item.participantId)}</span><strong>{labelCode(item.reason)}</strong></li>)}</ul></section>
      <section className="panel"><div className="panel-heading"><div><p className="eyebrow">Internal diagnostics</p><h2>Hard-rule exclusions</h2></div></div>{!rejectionSummary.length && <p className="empty-state">No candidate edge was rejected.</p>}<ul className="outcome-list">{rejectionSummary.map(([code, count]) => <li key={code}><span>{labelCode(code)}</span><strong>{count}</strong></li>)}</ul></section>
    </div>
  </>
}

function PairingCard({ pairing, canAdjust, onAdjust }: { pairing: Pairing; canAdjust: boolean; onAdjust: () => void }) {
  return <article className="pairing-card"><div className="score-ring"><strong>{pairing.score}</strong><small>%</small></div><div className="participant-pair"><span>{pairing.participantACode ?? shortId(pairing.participantA)}</span><i aria-hidden="true">◇</i><span>{pairing.participantBCode ?? shortId(pairing.participantB)}</span></div><ul>{pairing.safeReasons.map((reason) => <li key={reason}>{reason}</li>)}</ul><footer><span>{pairing.source === 'OVERRIDE' ? 'Administrator override' : 'Rule-based proposal'}</span>{canAdjust && pairing.pairingId && <button className="ghost-action" type="button" onClick={onAdjust}>Adjust pairing</button>}</footer></article>
}

function OverrideEditor({ run, pairing, onClose, onSaved }: { run: MatchingRun; pairing: Pairing; onClose: () => void; onSaved: (run: MatchingRun) => Promise<void> }) {
  const [message, setMessage] = useState('')
  const codeById = useMemo(() => {
    const codes = new Map<string, string>()
    ;[...(run.suggestions ?? []), ...(run.selections ?? [])].forEach((item) => { if (item.participantACode) codes.set(item.participantA, item.participantACode); if (item.participantBCode) codes.set(item.participantB, item.participantBCode) })
    run.unmatched?.forEach((item) => { if (item.participantCode) codes.set(item.participantId, item.participantCode) })
    return codes
  }, [run])
  const candidates = (run.candidates ?? []).filter((candidate) => candidate.eligible).sort((a, b) => (b.totalScore ?? 0) - (a.totalScore ?? 0))
  const mutation = useMutation({ mutationFn: (input: { removeSelectionId: string; participantA: string; participantB: string; reason: string }) => overridePairing(run, input), onSuccess: onSaved, onError: (problem) => setMessage((problem as unknown as Problem).detail ?? 'The pairing could not be adjusted.') })
  return <div className="modal-backdrop" role="presentation"><section className="override-dialog" role="dialog" aria-modal="true" aria-labelledby="override-title"><div className="panel-heading"><div><p className="eyebrow">Audited adjustment</p><h2 id="override-title">Replace {pairing.participantACode} + {pairing.participantBCode}</h2></div><button className="icon-button" type="button" onClick={onClose} aria-label="Close pairing adjustment">×</button></div><p>Hard eligibility and the minimum score remain mandatory. Participants already selected elsewhere cannot be reused.</p><form onSubmit={(event) => { event.preventDefault(); const data = new FormData(event.currentTarget); const [participantA, participantB] = String(data.get('candidate')).split('|'); mutation.mutate({ removeSelectionId: pairing.pairingId ?? '', participantA, participantB, reason: String(data.get('reason')).trim() }) }}><label>Eligible replacement<select name="candidate" required defaultValue={`${pairing.participantA}|${pairing.participantB}`}>{candidates.map((candidate) => <option key={`${candidate.participantA}-${candidate.participantB}`} value={`${candidate.participantA}|${candidate.participantB}`}>{codeById.get(candidate.participantA) ?? shortId(candidate.participantA)} + {codeById.get(candidate.participantB) ?? shortId(candidate.participantB)} · {candidate.totalScore}%</option>)}</select></label><label>Required audit reason<textarea name="reason" minLength={3} maxLength={500} required placeholder="Explain why this eligible replacement is appropriate." /></label><div className="dialog-actions"><button className="ghost-action" type="button" onClick={onClose}>Cancel</button><button className="primary-action" disabled={mutation.isPending}>{mutation.isPending ? 'Applying…' : 'Apply eligible replacement'}</button></div>{message && <p className="operation-message is-error" role="alert">{message}</p>}</form></section></div>
}

function nextStep(status: MatchingRun['status']) { if (status === 'GENERATED') return 'Inspect scores and safe reasons before opening review.'; if (status === 'UNDER_REVIEW') return 'Adjust only eligible pairs, then revalidate and lock.'; if (status === 'LOCKED') return 'Publish the immutable, member-safe results.'; if (status === 'PUBLISHED') return 'Members can now see only their own published match.'; return 'This run cannot be changed.' }
function formatDate(value: string, timeZone: string) { return new Intl.DateTimeFormat('en-LK', { dateStyle: 'medium', timeStyle: 'short', timeZone }).format(new Date(value)) }
function shortId(value: string) { return value.slice(0, 8).toUpperCase() }
function labelCode(value: string) { return value.toLowerCase().replaceAll('_', ' ').replace(/^./, (letter) => letter.toUpperCase()) }
