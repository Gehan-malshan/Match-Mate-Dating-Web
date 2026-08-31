package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/domain"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository              { return &Repository{pool: pool} }
func (r *Repository) Ready(ctx context.Context) error { return r.pool.Ping(ctx) }
func problem(status int, code, detail string) *domain.ProblemError {
	return &domain.ProblemError{Status: status, Code: code, Detail: detail}
}
func fact(ctx context.Context, tx pgx.Tx, eventType, aggregate, actor, correlation string, payload map[string]any, now time.Time) error {
	eventID := uuid.NewString()
	body, _ := json.Marshal(payload)
	_, err := tx.Exec(ctx, `INSERT INTO outbox(event_id,event_type,schema_version,occurred_at,aggregate_id,correlation_id,actor_id,routing_key,payload)VALUES($1,$2,1,$3,$4,$5,$6,$7,$8)`, eventID, eventType, now, aggregate, correlation, actor, "moderation."+eventType, body)
	return err
}
func audit(ctx context.Context, tx pgx.Tx, caseID, actor, operation, targetType, targetID, reason, correlation string, metadata map[string]any, now time.Time) error {
	body, _ := json.Marshal(metadata)
	_, err := tx.Exec(ctx, `INSERT INTO moderation_audit(audit_id,case_id,actor_id,operation,target_type,target_id,reason,metadata,correlation_id,occurred_at)VALUES($1,NULLIF($2,'')::uuid,$3,$4,$5,$6,$7,$8,$9,$10)`, uuid.NewString(), caseID, actor, operation, targetType, targetID, reason, body, correlation, now)
	return err
}
func (r *Repository) CreateReport(ctx context.Context, p domain.Principal, in domain.CreateReportInput, correlation string, now time.Time) (domain.Report, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Report{}, err
	}
	defer tx.Rollback(ctx)
	normalized := strings.ToLower(string(in.TargetType) + "|" + in.TargetID + "|" + string(in.Category) + "|" + strings.TrimSpace(in.Description))
	hash := sha256.Sum256([]byte(normalized))
	caseID, reportID := uuid.NewString(), uuid.NewString()
	severity := domain.SeverityMedium
	if in.Category == domain.CategorySafety {
		severity = domain.SeverityHigh
	}
	_, err = tx.Exec(ctx, `INSERT INTO moderation_case(case_id,status,severity,created_at,updated_at)VALUES($1,'OPEN',$2,$3,$3)`, caseID, severity, now)
	if err != nil {
		return domain.Report{}, err
	}
	role := "member"
	if p.HasRole("organizer") {
		role = "organizer"
	}
	_, err = tx.Exec(ctx, `INSERT INTO report(report_id,case_id,reporter_id,reporter_role,target_type,target_id,event_id,category,description,dedupe_hash,status,created_at,updated_at)VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::uuid,$8,$9,$10,'OPEN',$11,$11)`, reportID, caseID, p.Subject, role, in.TargetType, in.TargetID, in.EventID, in.Category, strings.TrimSpace(in.Description), hash[:], now)
	if err != nil {
		if strings.Contains(err.Error(), "report_reporter_id_dedupe_hash_key") {
			return domain.Report{}, problem(409, "DUPLICATE_REPORT", "An equivalent report was already submitted")
		}
		return domain.Report{}, err
	}
	for _, e := range in.Evidence {
		_, err = tx.Exec(ctx, `INSERT INTO evidence_reference(evidence_id,report_id,reference,media_type,sha256,retain_until,created_at)VALUES($1,$2,$3,$4,$5,$6,$7)`, uuid.NewString(), reportID, e.Reference, e.MediaType, strings.ToLower(e.SHA256), e.RetainUntil, now)
		if err != nil {
			return domain.Report{}, err
		}
	}
	if err = audit(ctx, tx, caseID, p.Subject, "REPORT_SUBMITTED", string(in.TargetType), in.TargetID, "Member submitted report", correlation, map[string]any{"category": in.Category, "evidenceCount": len(in.Evidence)}, now); err != nil {
		return domain.Report{}, err
	}
	if err = fact(ctx, tx, "ReportSubmitted", reportID, p.Subject, correlation, map[string]any{"reportId": reportID, "caseId": caseID, "targetType": in.TargetType, "targetId": in.TargetID, "category": in.Category, "severity": severity}, now); err != nil {
		return domain.Report{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Report{}, err
	}
	return domain.Report{ID: reportID, CaseID: caseID, TargetType: in.TargetType, TargetID: in.TargetID, Category: in.Category, Status: domain.ReportOpen, Severity: severity, CreatedAt: now, UpdatedAt: now}, nil
}
func next(t time.Time, id string) string {
	raw, _ := json.Marshal(map[string]any{"t": t, "id": id})
	return base64.RawURLEncoding.EncodeToString(raw)
}
func scanReport(row pgx.Row) (domain.Report, error) {
	var value domain.Report
	err := row.Scan(&value.ID, &value.CaseID, &value.TargetType, &value.TargetID, &value.Category, &value.Description, &value.Status, &value.Severity, &value.CreatedAt, &value.UpdatedAt)
	return value, err
}
func (r *Repository) ListOwnReports(ctx context.Context, account string, limit int, cursorTime time.Time, cursorID string) (domain.Page[domain.Report], error) {
	rows, err := r.pool.Query(ctx, `SELECT r.report_id,r.case_id,r.target_type,r.target_id,r.category,''::text,r.status,c.severity,r.created_at,r.updated_at FROM report r JOIN moderation_case c USING(case_id) WHERE r.reporter_id=$1 AND (r.created_at,r.report_id)<($2,$3::uuid) ORDER BY r.created_at DESC,r.report_id DESC LIMIT $4`, account, cursorTime, cursorID, limit+1)
	if err != nil {
		return domain.Page[domain.Report]{}, err
	}
	defer rows.Close()
	page := domain.Page[domain.Report]{Items: []domain.Report{}, Limit: limit}
	for rows.Next() {
		item, scanErr := scanReport(rows)
		if scanErr != nil {
			return page, scanErr
		}
		page.Items = append(page.Items, item)
	}
	if len(page.Items) > limit {
		last := page.Items[limit-1]
		page.NextCursor = next(last.CreatedAt, last.ID)
		page.Items = page.Items[:limit]
	}
	return page, rows.Err()
}
func scanCase(row pgx.Row) (domain.Case, error) {
	var value domain.Case
	err := row.Scan(&value.ID, &value.Status, &value.Severity, &value.AssigneeID, &value.SLAAt, &value.Version, &value.CreatedAt, &value.UpdatedAt)
	return value, err
}
func (r *Repository) ListCases(ctx context.Context, limit int, cursorTime time.Time, cursorID string) (domain.Page[domain.Case], error) {
	rows, err := r.pool.Query(ctx, `SELECT case_id,status,severity,COALESCE(assignee_id::text,''),sla_at,version,created_at,updated_at FROM moderation_case WHERE (updated_at,case_id)<($1,$2::uuid) ORDER BY updated_at DESC,case_id DESC LIMIT $3`, cursorTime, cursorID, limit+1)
	if err != nil {
		return domain.Page[domain.Case]{}, err
	}
	defer rows.Close()
	page := domain.Page[domain.Case]{Items: []domain.Case{}, Limit: limit}
	for rows.Next() {
		item, e := scanCase(rows)
		if e != nil {
			return page, e
		}
		page.Items = append(page.Items, item)
	}
	if len(page.Items) > limit {
		last := page.Items[limit-1]
		page.NextCursor = next(last.UpdatedAt, last.ID)
		page.Items = page.Items[:limit]
	}
	return page, rows.Err()
}
func (r *Repository) GetCase(ctx context.Context, id string) (domain.Case, error) {
	value, err := scanCase(r.pool.QueryRow(ctx, `SELECT case_id,status,severity,COALESCE(assignee_id::text,''),sla_at,version,created_at,updated_at FROM moderation_case WHERE case_id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return value, problem(404, "MODERATION_CASE_NOT_FOUND", "Moderation case was not found")
	}
	if err != nil {
		return value, err
	}
	reports, err := r.pool.Query(ctx, `SELECT r.report_id,r.case_id,r.target_type,r.target_id,r.category,r.description,r.status,c.severity,r.created_at,r.updated_at FROM report r JOIN moderation_case c USING(case_id) WHERE r.case_id=$1 ORDER BY r.created_at`, id)
	if err != nil {
		return value, err
	}
	defer reports.Close()
	for reports.Next() {
		item, e := scanReport(reports)
		if e != nil {
			return value, e
		}
		value.Reports = append(value.Reports, item)
	}
	actions, err := r.pool.Query(ctx, `SELECT action_id,case_id,target_type,target_id,action_class,scope,reason,version,effective_at,expires_at,state,created_at FROM moderation_action WHERE case_id=$1 ORDER BY created_at`, id)
	if err != nil {
		return value, err
	}
	defer actions.Close()
	for actions.Next() {
		var a domain.Action
		if err = actions.Scan(&a.ID, &a.CaseID, &a.TargetType, &a.TargetID, &a.Class, &a.Scope, &a.Reason, &a.Version, &a.EffectiveAt, &a.ExpiresAt, &a.State, &a.CreatedAt); err != nil {
			return value, err
		}
		value.Actions = append(value.Actions, a)
	}
	appeals, err := r.pool.Query(ctx, `SELECT ap.appeal_id,ap.action_id,ap.appellant_id,ap.reason,ap.state,COALESCE(ap.decision_reason,''),ap.created_at,ap.decided_at FROM appeal ap JOIN moderation_action a USING(action_id) WHERE a.case_id=$1 ORDER BY ap.created_at`, id)
	if err != nil {
		return value, err
	}
	defer appeals.Close()
	for appeals.Next() {
		var a domain.Appeal
		if err = appeals.Scan(&a.ID, &a.ActionID, &a.AppellantID, &a.Reason, &a.State, &a.DecisionReason, &a.CreatedAt, &a.DecidedAt); err != nil {
			return value, err
		}
		value.Appeals = append(value.Appeals, a)
	}
	return value, nil
}
func (r *Repository) RecordCaseView(ctx context.Context, caseID string, p domain.Principal, correlation string, now time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT true FROM moderation_case WHERE case_id=$1`, caseID).Scan(&exists); errors.Is(err, pgx.ErrNoRows) {
		return problem(404, "MODERATION_CASE_NOT_FOUND", "Moderation case was not found")
	}
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, caseID, p.Subject, "CASE_VIEWED", "CASE", caseID, "Privileged case view", correlation, map[string]any{}, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) AssignCase(ctx context.Context, caseID, assignee string, p domain.Principal, reason, correlation string, now time.Time) (domain.Case, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Case{}, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE moderation_case SET assignee_id=$2,status=CASE WHEN status='OPEN' THEN 'TRIAGED' ELSE status END,version=version+1,updated_at=$3 WHERE case_id=$1 AND status NOT IN('ACTIONED','DISMISSED')`, caseID, assignee, now)
	if err != nil {
		return domain.Case{}, err
	}
	if tag.RowsAffected() == 0 {
		return domain.Case{}, problem(409, "CASE_NOT_ASSIGNABLE", "Case cannot be assigned in its current state")
	}
	if err = audit(ctx, tx, caseID, p.Subject, "CASE_ASSIGNED", "CASE", caseID, reason, correlation, map[string]any{"assigneeId": assignee}, now); err != nil {
		return domain.Case{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Case{}, err
	}
	return r.GetCase(ctx, caseID)
}
func (r *Repository) UpdateCaseStatus(ctx context.Context, caseID string, next domain.ReportStatus, p domain.Principal, reason, correlation string, now time.Time) (domain.Case, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Case{}, err
	}
	defer tx.Rollback(ctx)
	var current domain.ReportStatus
	if err = tx.QueryRow(ctx, `SELECT status FROM moderation_case WHERE case_id=$1 FOR UPDATE`, caseID).Scan(&current); errors.Is(err, pgx.ErrNoRows) {
		return domain.Case{}, problem(404, "MODERATION_CASE_NOT_FOUND", "Moderation case was not found")
	}
	if err != nil {
		return domain.Case{}, err
	}
	if current == next {
		_ = tx.Rollback(ctx)
		return r.GetCase(ctx, caseID)
	}
	allowed := next == domain.ReportInvestigating && current == domain.ReportTriaged
	allowed = allowed || next == domain.ReportDismissed && current == domain.ReportInvestigating
	if !allowed {
		return domain.Case{}, problem(409, "CASE_STATUS_TRANSITION_INVALID", "Case cannot move to the requested state")
	}
	if _, err = tx.Exec(ctx, `UPDATE moderation_case SET status=$2,version=version+1,updated_at=$3 WHERE case_id=$1`, caseID, next, now); err != nil {
		return domain.Case{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE report SET status=$2,updated_at=$3 WHERE case_id=$1`, caseID, next, now); err != nil {
		return domain.Case{}, err
	}
	operation := "CASE_INVESTIGATION_STARTED"
	if next == domain.ReportDismissed {
		operation = "CASE_DISMISSED"
	}
	if err = audit(ctx, tx, caseID, p.Subject, operation, "CASE", caseID, reason, correlation, map[string]any{"previousStatus": current, "status": next}, now); err != nil {
		return domain.Case{}, err
	}
	if next == domain.ReportDismissed {
		if err = fact(ctx, tx, "ModerationCaseDismissed", caseID, p.Subject, correlation, map[string]any{"caseId": caseID, "state": next}, now); err != nil {
			return domain.Case{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Case{}, err
	}
	return r.GetCase(ctx, caseID)
}
func (r *Repository) CreateAction(ctx context.Context, caseID string, a domain.Action, p domain.Principal, correlation string, now time.Time) (domain.Action, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return a, err
	}
	defer tx.Rollback(ctx)
	var status string
	if err = tx.QueryRow(ctx, `SELECT status FROM moderation_case WHERE case_id=$1 FOR UPDATE`, caseID).Scan(&status); errors.Is(err, pgx.ErrNoRows) {
		return a, problem(404, "MODERATION_CASE_NOT_FOUND", "Moderation case was not found")
	}
	if err != nil {
		return a, err
	}
	if status != string(domain.ReportInvestigating) && status != string(domain.ReportActioned) {
		return a, problem(409, "CASE_NOT_ACTIONABLE", "Only an investigating case can receive its first action")
	}
	if err = tx.QueryRow(ctx, `SELECT COALESCE(max(version),0)+1 FROM moderation_action WHERE target_type=$1 AND target_id=$2 AND action_class=$3 AND scope=$4`, a.TargetType, a.TargetID, a.Class, a.Scope).Scan(&a.Version); err != nil {
		return a, err
	}
	a.ID = uuid.NewString()
	a.CaseID = caseID
	a.State = "ACTIVE"
	a.CreatedAt = now
	_, err = tx.Exec(ctx, `INSERT INTO moderation_action(action_id,case_id,target_type,target_id,action_class,scope,reason,actor_id,version,effective_at,expires_at,state,created_at)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'ACTIVE',$12)`, a.ID, caseID, a.TargetType, a.TargetID, a.Class, a.Scope, a.Reason, p.Subject, a.Version, a.EffectiveAt, a.ExpiresAt, now)
	if err != nil {
		if strings.Contains(err.Error(), "moderation_action_active_idx") {
			return a, problem(409, "ACTION_ALREADY_ACTIVE", "An equivalent safety action is already active")
		}
		return a, err
	}
	_, err = tx.Exec(ctx, `UPDATE moderation_case SET status='ACTIONED',version=version+1,updated_at=$2 WHERE case_id=$1`, caseID, now)
	if err != nil {
		return a, err
	}
	if _, err = tx.Exec(ctx, `UPDATE report SET status='ACTIONED',updated_at=$2 WHERE case_id=$1`, caseID, now); err != nil {
		return a, err
	}
	if err = audit(ctx, tx, caseID, p.Subject, "ACTION_APPLIED", string(a.TargetType), a.TargetID, a.Reason, correlation, map[string]any{"actionId": a.ID, "actionClass": a.Class, "scope": a.Scope, "version": a.Version, "expiresAt": a.ExpiresAt}, now); err != nil {
		return a, err
	}
	payload := map[string]any{"actionId": a.ID, "caseId": caseID, "targetType": a.TargetType, "targetId": a.TargetID, "actionClass": a.Class, "scope": a.Scope, "version": a.Version, "state": "ACTIVE", "effectiveAt": a.EffectiveAt, "expiresAt": a.ExpiresAt}
	if err = fact(ctx, tx, "ModerationActionApplied", a.ID, p.Subject, correlation, payload, now); err != nil {
		return a, err
	}
	if err = tx.Commit(ctx); err != nil {
		return a, err
	}
	return a, nil
}
func (r *Repository) CreateAppeal(ctx context.Context, actionID, reason string, p domain.Principal, correlation string, now time.Time) (domain.Appeal, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Appeal{}, err
	}
	defer tx.Rollback(ctx)
	var caseID, targetID string
	if err = tx.QueryRow(ctx, `SELECT case_id,target_id FROM moderation_action WHERE action_id=$1 AND state='ACTIVE'`, actionID).Scan(&caseID, &targetID); errors.Is(err, pgx.ErrNoRows) {
		return domain.Appeal{}, problem(404, "ACTION_NOT_APPEALABLE", "Active action was not found")
	}
	if err != nil {
		return domain.Appeal{}, err
	}
	if targetID != p.Subject {
		return domain.Appeal{}, problem(403, "APPEAL_FORBIDDEN", "Only the affected account may appeal this action")
	}
	appeal := domain.Appeal{ID: uuid.NewString(), ActionID: actionID, AppellantID: p.Subject, Reason: strings.TrimSpace(reason), State: "APPEALED", CreatedAt: now}
	_, err = tx.Exec(ctx, `INSERT INTO appeal(appeal_id,action_id,appellant_id,reason,state,created_at)VALUES($1,$2,$3,$4,'APPEALED',$5)`, appeal.ID, actionID, p.Subject, appeal.Reason, now)
	if err != nil {
		if strings.Contains(err.Error(), "appeal_action_id_appellant_id_key") {
			return appeal, problem(409, "APPEAL_ALREADY_EXISTS", "This action already has an appeal")
		}
		return appeal, err
	}
	if err = audit(ctx, tx, caseID, p.Subject, "APPEAL_SUBMITTED", "ACTION", actionID, "Member submitted appeal", correlation, map[string]any{"appealId": appeal.ID}, now); err != nil {
		return appeal, err
	}
	if err = fact(ctx, tx, "AppealSubmitted", appeal.ID, p.Subject, correlation, map[string]any{"appealId": appeal.ID, "actionId": actionID, "targetId": targetID, "state": "APPEALED"}, now); err != nil {
		return appeal, err
	}
	if err = tx.Commit(ctx); err != nil {
		return appeal, err
	}
	return appeal, nil
}
func (r *Repository) DecideAppeal(ctx context.Context, appealID, decision string, p domain.Principal, reason, correlation string, now time.Time) (domain.Appeal, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Appeal{}, err
	}
	defer tx.Rollback(ctx)
	var a domain.Appeal
	var caseID, targetType, targetID, class, scope string
	var version int64
	err = tx.QueryRow(ctx, `SELECT ap.appeal_id,ap.action_id,ap.appellant_id,ap.reason,ap.created_at,a.case_id,a.target_type,a.target_id,a.action_class,a.scope,a.version FROM appeal ap JOIN moderation_action a USING(action_id) WHERE ap.appeal_id=$1 AND ap.state='APPEALED' FOR UPDATE OF ap,a`, appealID).Scan(&a.ID, &a.ActionID, &a.AppellantID, &a.Reason, &a.CreatedAt, &caseID, &targetType, &targetID, &class, &scope, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, problem(404, "APPEAL_NOT_PENDING", "Pending appeal was not found")
	}
	if err != nil {
		return a, err
	}
	a.State = decision
	a.DecisionReason = strings.TrimSpace(reason)
	a.DecidedAt = &now
	_, err = tx.Exec(ctx, `UPDATE appeal SET state=$2,decision_actor_id=$3,decision_reason=$4,decided_at=$5 WHERE appeal_id=$1`, appealID, decision, p.Subject, a.DecisionReason, now)
	if err != nil {
		return a, err
	}
	eventType := "AppealUpheld"
	if decision == "REVERSED" {
		eventType = "ModerationActionReversed"
		_, err = tx.Exec(ctx, `UPDATE moderation_action SET state='REVERSED' WHERE action_id=$1`, a.ActionID)
		if err != nil {
			return a, err
		}
	}
	if err = audit(ctx, tx, caseID, p.Subject, "APPEAL_"+decision, "APPEAL", appealID, a.DecisionReason, correlation, map[string]any{"actionId": a.ActionID}, now); err != nil {
		return a, err
	}
	payload := map[string]any{"appealId": a.ID, "actionId": a.ActionID, "targetType": targetType, "targetId": targetID, "actionClass": class, "scope": scope, "version": version, "state": decision}
	if err = fact(ctx, tx, eventType, a.ActionID, p.Subject, correlation, payload, now); err != nil {
		return a, err
	}
	if err = tx.Commit(ctx); err != nil {
		return a, err
	}
	return a, nil
}
func (r *Repository) ExpireActions(ctx context.Context, now time.Time, limit int) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT action_id,case_id,target_type,target_id,action_class,scope,version FROM moderation_action WHERE state='ACTIVE' AND expires_at<=$1 ORDER BY expires_at LIMIT $2 FOR UPDATE SKIP LOCKED`, now, limit)
	if err != nil {
		return 0, err
	}
	type item struct {
		id, caseID, targetType, targetID, class, scope string
		version                                        int64
	}
	var items []item
	for rows.Next() {
		var i item
		if err = rows.Scan(&i.id, &i.caseID, &i.targetType, &i.targetID, &i.class, &i.scope, &i.version); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, i)
	}
	rows.Close()
	for _, i := range items {
		if _, err = tx.Exec(ctx, `UPDATE moderation_action SET state='EXPIRED' WHERE action_id=$1`, i.id); err != nil {
			return 0, err
		}
		if err = audit(ctx, tx, i.caseID, "00000000-0000-0000-0000-000000000000", "ACTION_EXPIRED", i.targetType, i.targetID, "Scheduled expiry", "expiry-worker", map[string]any{"actionId": i.id}, now); err != nil {
			return 0, err
		}
		if err = fact(ctx, tx, "ModerationActionExpired", i.id, "00000000-0000-0000-0000-000000000000", "expiry-worker", map[string]any{"actionId": i.id, "targetType": i.targetType, "targetId": i.targetID, "actionClass": i.class, "scope": i.scope, "version": i.version, "state": "EXPIRED"}, now); err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(items), nil
}
func (r *Repository) ClaimOutbox(ctx context.Context, limit int) ([]store.OutboxRecord, error) {
	rows, err := r.pool.Query(ctx, `WITH picked AS(SELECT event_id FROM outbox WHERE published_at IS NULL AND(claimed_at IS NULL OR claimed_at<now()-interval '1 minute')ORDER BY occurred_at LIMIT $1 FOR UPDATE SKIP LOCKED)UPDATE outbox o SET claimed_at=now(),attempts=attempts+1 FROM picked WHERE o.event_id=picked.event_id RETURNING o.event_id::text,o.routing_key,jsonb_build_object('eventId',o.event_id,'eventType',o.event_type,'schemaVersion',o.schema_version,'occurredAt',o.occurred_at,'aggregateId',o.aggregate_id,'correlationId',o.correlation_id,'causationId',o.causation_id,'actorId',o.actor_id,'payload',o.payload)::text`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []store.OutboxRecord
	for rows.Next() {
		var record store.OutboxRecord
		var body string
		if err = rows.Scan(&record.ID, &record.RoutingKey, &body); err != nil {
			return nil, err
		}
		record.Body = []byte(body)
		records = append(records, record)
	}
	return records, rows.Err()
}
func (r *Repository) MarkOutboxPublished(ctx context.Context, id string, at time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE outbox SET published_at=$2,claimed_at=NULL WHERE event_id=$1`, id, at)
	return err
}
