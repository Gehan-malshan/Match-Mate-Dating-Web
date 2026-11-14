import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AdminAccount, archiveEvent, createEvent, EventInput, listArchivedEvents, ManagedEvent, Problem, restoreEvent, transitionEvent, updateEvent, uploadEventImage } from '../lib/api'
import { prepareEventImage } from '../lib/event-image'

export function EventsView({ account, events, loading, error, onViewRegistrations }: { account: AdminAccount; events: ManagedEvent[]; loading: boolean; error: boolean; onViewRegistrations: (eventId: string) => void }) {
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState<ManagedEvent | 'new'>()
  const [showDeleted, setShowDeleted] = useState(false)
  const [message, setMessage] = useState('')
  const deletedQuery = useQuery({ queryKey: ['admin-archived-events'], queryFn: listArchivedEvents, enabled: showDeleted })
  const visibleEvents = showDeleted ? deletedQuery.data?.items ?? [] : events
  const visibleLoading = showDeleted ? deletedQuery.isPending : loading
  const visibleError = showDeleted ? deletedQuery.isError : error

  async function transition(event: ManagedEvent, action: 'publish' | 'open-registration' | 'close-registration' | 'cancel') {
    if (action === 'cancel') {
      if (!window.confirm(`Cancel “${event.name}”? This changes its public status.`)) return
    }
    setMessage('Applying the versioned event change…')
    try {
      await transitionEvent(event, action)
      await queryClient.invalidateQueries({ queryKey: ['admin-events'] })
      setMessage('Event lifecycle updated.')
    } catch (problem) {
      setMessage((problem as unknown as Problem).detail ?? 'The event could not be updated.')
    }
  }

  async function setDeleted(event: ManagedEvent, restore: boolean) {
    if (!window.confirm(restore ? `Restore “${event.name}” to the active event list?` : `Delete “${event.name}” from the active event list? The event and its booking history will be retained and you can restore it later.`)) return
    setMessage(`${restore ? 'Restoring' : 'Deleting'} event…`)
    try {
      if (restore) await restoreEvent(event)
      else await archiveEvent(event)
      await Promise.all([queryClient.invalidateQueries({ queryKey: ['admin-events'] }), queryClient.invalidateQueries({ queryKey: ['admin-archived-events'] })])
      setMessage(restore ? 'Event restored.' : 'Event removed from the active list. Use Show deleted to restore it.')
    } catch (problem) {
      setMessage((problem as unknown as Problem).detail ?? `The event could not be ${restore ? 'restored' : 'deleted'}.`)
    }
  }

  return (
    <section className="admin-view">
      <header className="view-heading">
        <div><p className="eyebrow">Event manager</p><h1>Shape every <em>encounter.</em></h1><p>Create private drafts, validate the schedule, and deliberately move approved events into public registration.</p></div>
        <button className="primary-action" type="button" onClick={() => setEditing('new')}>Create new event <span aria-hidden="true">＋</span></button>
      </header>

      <div className="event-policy-banner"><span aria-hidden="true">⌾</span><div><strong>Administrator-only event controls</strong><p>Delete removes a draft or cancelled event from the active list without erasing booking history. Cancel a live event before deleting it.</p></div></div>

      {editing && <EventEditor account={account} event={editing === 'new' ? undefined : editing} onClose={() => setEditing(undefined)} />}

      <section className="panel event-catalog">
        <div className="panel-heading"><div><p className="eyebrow">Event inventory</p><h2>{showDeleted ? 'Deleted events' : 'Managed events'}</h2></div><button className="ghost-action" type="button" onClick={() => { setShowDeleted(!showDeleted); setMessage('') }}>{showDeleted ? 'Show active' : 'Show deleted'}</button></div>
        {visibleLoading && <p className="empty-state" role="status">Loading events…</p>}
        {visibleError && <p className="empty-state is-error" role="alert">The Event Service is unavailable or this session no longer has administrator access.</p>}
        {!visibleLoading && !visibleError && visibleEvents.length === 0 && <p className="empty-state">{showDeleted ? 'No deleted events.' : 'No active events exist yet. Create an administrator-owned draft to begin.'}</p>}
        <div className="event-grid">
          {visibleEvents.map((event) => (
            <article className="event-card" key={event.eventId}>
              <div className="event-card-glow" aria-hidden="true" />
              <div className="event-card-top"><span className={`status-badge status-${event.status.toLowerCase()}`}>{event.status.replaceAll('_', ' ')}</span><small>v{event.version}</small></div>
              <p className="event-date">{formatDate(event.startsAt, event.timeZone)}</p>
              <h3>{event.name}</h3><p>{event.description || 'No public description has been added yet.'}</p>
              <dl><div><dt>Location</dt><dd>{event.broadLocation}</dd></div><div><dt>Capacity</dt><dd>{event.configuredCapacity}</dd></div><div><dt>Ticket</dt><dd>{event.currency} {event.price}</dd></div><div><dt>Payment</dt><dd>{event.paymentOptions.replaceAll('_',' ').toLowerCase()}</dd></div><div><dt>Image</dt><dd>{event.imageVersion ? 'Attached' : 'Default artwork'}</dd></div></dl>
              <div className="card-actions">
                <button className="ghost-action" type="button" onClick={() => onViewRegistrations(event.eventId)}>View registrations</button>
                {showDeleted ? <button className="small-primary" type="button" onClick={() => void setDeleted(event, true)}>Restore</button> : <>
                  {event.status === 'DRAFT' && <><button className="ghost-action" type="button" onClick={() => setEditing(event)}>Edit draft</button><button className="small-primary" type="button" onClick={() => void transition(event, 'publish')}>Publish</button></>}
                  {event.status === 'PUBLISHED' && <button className="small-primary" type="button" onClick={() => void transition(event, 'open-registration')}>Open registration</button>}
                  {event.status === 'REGISTRATION_OPEN' && <button className="small-primary" type="button" onClick={() => void transition(event, 'close-registration')}>Close registration</button>}
                  {event.status !== 'CANCELLED' && <button className="danger-action" type="button" onClick={() => void transition(event, 'cancel')}>Cancel</button>}
                  <button className="danger-action" type="button" disabled={event.status !== 'DRAFT' && event.status !== 'CANCELLED'} title={event.status !== 'DRAFT' && event.status !== 'CANCELLED' ? 'Cancel this event before deleting it' : 'Remove this event from the active list'} onClick={() => void setDeleted(event, false)}>Delete</button>
                </>}
              </div>
            </article>
          ))}
        </div>
        {message && <p className="operation-message" role="status">{message}</p>}
      </section>
    </section>
  )
}

function EventEditor({ account, event, onClose }: { account: AdminAccount; event?: ManagedEvent; onClose: () => void }) {
  const queryClient = useQueryClient()
  const [message, setMessage] = useState('')
  const [imageFile, setImageFile] = useState<File | null>(null)
  const [preview, setPreview] = useState('')
  useEffect(() => {
    if (!imageFile) { setPreview(''); return }
    const url = URL.createObjectURL(imageFile)
    setPreview(url)
    return () => URL.revokeObjectURL(url)
  }, [imageFile])
  const source = event ?? initialEvent(account.id)
  const mutation = useMutation({
    mutationFn: async (input: EventInput) => {
      let encoded = ''
      if (imageFile) {
        try { encoded = await prepareEventImage(imageFile) }
        catch (error) { throw { detail: error instanceof Error ? error.message : 'Image could not be prepared.' } }
      }
      const saved = event ? await updateEvent(event.eventId, input, event.version) : await createEvent(input)
      if (encoded) {
        try { await uploadEventImage(saved.eventId, encoded) }
        catch { await queryClient.invalidateQueries({ queryKey: ['admin-events'] }); throw { detail: 'The draft was saved, but the image upload failed. Close this editor and reopen the draft to retry.' } }
      }
      return saved
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['admin-events'] })
      onClose()
    },
    onError: (problem) => setMessage(formatProblem(problem as unknown as Problem)),
  })

  return (
    <section className="panel event-editor" aria-labelledby="event-editor-title">
      <div className="panel-heading"><div><p className="eyebrow">{event ? 'Draft editor' : 'New administrator event'}</p><h2 id="event-editor-title">{event ? event.name : 'Event essence'}</h2></div><button className="icon-button" type="button" onClick={onClose} aria-label="Close event editor">×</button></div>
      <form onSubmit={(formEvent) => {
        formEvent.preventDefault()
        setMessage('')
        const data = new FormData(formEvent.currentTarget)
        mutation.mutate({
          organizerId: account.id,
          name: String(data.get('name')).trim(),
          description: String(data.get('description')).trim(),
          venueName: String(data.get('venueName')).trim(),
          broadLocation: String(data.get('broadLocation')).trim(),
          timeZone: String(data.get('timeZone')).trim(),
          startsAt: new Date(String(data.get('startsAt'))).toISOString(),
          endsAt: new Date(String(data.get('endsAt'))).toISOString(),
          registrationOpensAt: new Date(String(data.get('registrationOpensAt'))).toISOString(),
          registrationClosesAt: new Date(String(data.get('registrationClosesAt'))).toISOString(),
          price: String(data.get('price')).trim(),
          currency: String(data.get('currency')).trim().toUpperCase(),
          paymentOptions: String(data.get('paymentOptions')) as EventInput['paymentOptions'],
          configuredCapacity: Number(data.get('configuredCapacity')),
          matchingRulesetVersion: String(data.get('matchingRulesetVersion')).trim(),
        })
      }}>
        <fieldset><legend>Event essence</legend><div className="field-grid"><label>Event name<input name="name" defaultValue={source.name} minLength={3} maxLength={120} required /></label><label>Broad public location<input name="broadLocation" defaultValue={source.broadLocation} required /></label><label className="wide">Public description<textarea name="description" defaultValue={source.description} maxLength={2000} /></label><label>Exact venue <small>Admin only</small><input name="venueName" defaultValue={source.venueName} /></label><label>Venue time zone<input name="timeZone" defaultValue={source.timeZone} required /></label></div></fieldset>
        <fieldset><legend>Event image</legend><div className="field-grid"><label className="wide">Upload event photo <small>JPEG, PNG, or WebP. It will be resized and saved as JPEG without embedded metadata.</small><input name="eventImage" type="file" accept="image/jpeg,image/png,image/webp" onChange={(change) => setImageFile(change.currentTarget.files?.[0] ?? null)} /></label><p className="event-image-note">Use only artwork you have permission to publish. Avoid identifiable members and private venue details.</p>{preview && <img className="event-image-preview" src={preview} alt="Preview of selected event artwork" />}{!preview && !!event?.imageVersion && <p className="event-image-note">An image is attached. Choose a new file to replace it while this event is a draft.</p>}</div></fieldset>
        <fieldset><legend>Journey logistics</legend><div className="field-grid"><label>Starts<input name="startsAt" type="datetime-local" defaultValue={localDate(source.startsAt)} required /></label><label>Ends<input name="endsAt" type="datetime-local" defaultValue={localDate(source.endsAt)} required /></label><label>Registration opens<input name="registrationOpensAt" type="datetime-local" defaultValue={localDate(source.registrationOpensAt)} required /></label><label>Registration closes<input name="registrationClosesAt" type="datetime-local" defaultValue={localDate(source.registrationClosesAt)} required /></label></div></fieldset>
        <fieldset><legend>Capacity and matching</legend><div className="field-grid three"><label>Ticket price<input name="price" inputMode="decimal" defaultValue={source.price} placeholder="3500.00" required /></label><label>Currency<input name="currency" defaultValue={source.currency} minLength={3} maxLength={3} required /></label><label>Configured capacity<input name="configuredCapacity" type="number" min={1} max={10000} defaultValue={source.configuredCapacity} required /></label><label>How members pay<select name="paymentOptions" defaultValue={source.paymentOptions} required><option value="AT_VENUE">At the venue</option><option value="ONLINE">Online with PayHere</option><option value="BOTH">Either option</option></select></label><label className="wide">Matching ruleset<input name="matchingRulesetVersion" defaultValue={source.matchingRulesetVersion} required /></label></div></fieldset>
        <div className="editor-footer"><p>Saving creates a private draft. Publication is a separate action.</p><div><button className="ghost-action" type="button" onClick={onClose}>Cancel</button><button className="primary-action" disabled={mutation.isPending}>{mutation.isPending ? 'Saving…' : 'Save private draft'}</button></div></div>
        {message && <p className="operation-message is-error" role="alert">{message}</p>}
      </form>
    </section>
  )
}

function initialEvent(organizerId: string): EventInput {
  const start = addDays(new Date(), 30, 19)
  const end = new Date(start.getTime() + 3 * 60 * 60 * 1000)
  const opens = addDays(new Date(), 1, 9)
  const closes = new Date(start.getTime() - 24 * 60 * 60 * 1000)
  return { organizerId, name: '', description: '', venueName: '', broadLocation: 'Colombo', timeZone: 'Asia/Colombo', startsAt: start.toISOString(), endsAt: end.toISOString(), registrationOpensAt: opens.toISOString(), registrationClosesAt: closes.toISOString(), price: '3500.00', currency: 'LKR', paymentOptions: 'AT_VENUE', configuredCapacity: 40, matchingRulesetVersion: 'prototype-v1' }
}

function addDays(date: Date, days: number, hour: number) { const value = new Date(date); value.setDate(value.getDate() + days); value.setHours(hour, 0, 0, 0); return value }
function localDate(value: string) { const date = new Date(value); return new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 16) }
function formatDate(value: string, timeZone: string) { return new Intl.DateTimeFormat('en-LK', { dateStyle: 'medium', timeStyle: 'short', timeZone }).format(new Date(value)) }
function formatProblem(problem: Problem) { const fields = Object.values(problem.fieldErrors ?? {}); return fields.length ? `${problem.detail}: ${fields.join(', ')}` : problem.detail }
