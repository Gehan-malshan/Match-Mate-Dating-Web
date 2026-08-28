package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gehan-malshan/matchmate/account-service/internal/domain"
	"github.com/gehan-malshan/matchmate/account-service/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository             { return &Repository{pool} }
func (r *Repository) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }

func (r *Repository) Register(ctx context.Context, in domain.RegisterInput, passwordHash string, verificationHash []byte, verificationExpiry time.Time, event domain.Event) (domain.Me, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Me{}, err
	}
	defer tx.Rollback(ctx)
	now := event.OccurredAt
	id := uuid.NewString()
	_, err = tx.Exec(ctx, `INSERT INTO account(account_id,normalized_email,status,verification_state,created_at,updated_at) VALUES($1,$2,'ACTIVE','PENDING',$3,$3)`, id, in.Email, now)
	if isUnique(err) {
		return domain.Me{}, store.ErrConflict
	}
	if err != nil {
		return domain.Me{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO credential(account_id,password_hash,changed_at) VALUES($1,$2,$3)`, id, passwordHash, now); err != nil {
		return domain.Me{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO consent_record(consent_id,account_id,consent_type,policy_version,granted_at) VALUES($1,$2,'privacy-and-terms',$3,$4)`, uuid.NewString(), id, in.ConsentVersion, now); err != nil {
		return domain.Me{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO role_assignment(account_id,role,granted_at) VALUES($1,'member',$2)`, id, now); err != nil {
		return domain.Me{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO profile(account_id,nickname,date_of_birth,visibility,approval_state,updated_at) VALUES($1,$2,$3,'PRIVATE','PENDING',$4)`, id, in.Nickname, in.DateOfBirth, now); err != nil {
		return domain.Me{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO email_verification_token(token_hash,account_id,expires_at,created_at) VALUES($1,$2,$3,$4)`, verificationHash, id, verificationExpiry, now); err != nil {
		return domain.Me{}, err
	}
	event.AggregateID = id
	event.Payload = map[string]any{"accountId": id, "verificationState": "PENDING"}
	if err = insertEvent(ctx, tx, event); err != nil {
		return domain.Me{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Me{}, err
	}
	return domain.Me{Account: domain.Account{ID: id, Email: in.Email, Status: domain.AccountActive, Verification: domain.VerificationPending, Roles: []string{"member"}, CreatedAt: now, UpdatedAt: now}, Profile: domain.Profile{AccountID: id, Nickname: in.Nickname, DateOfBirth: in.DateOfBirth, Visibility: domain.VisibilityPrivate, Approval: domain.ApprovalPending, Version: 1, UpdatedAt: now}}, nil
}

func (r *Repository) VerifyEmail(ctx context.Context, hash []byte, now time.Time, event domain.Event) (domain.Account, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Account{}, err
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(ctx, `SELECT account_id FROM email_verification_token WHERE token_hash=$1 AND consumed_at IS NULL AND expires_at>$2 FOR UPDATE`, hash, now).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Account{}, store.ErrInvalidToken
	}
	if err != nil {
		return domain.Account{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE email_verification_token SET consumed_at=$2 WHERE token_hash=$1`, hash, now); err != nil {
		return domain.Account{}, err
	}
	tag, err := tx.Exec(ctx, `UPDATE account SET verification_state='VERIFIED',updated_at=$2 WHERE account_id=$1 AND status='ACTIVE'`, id, now)
	if err != nil {
		return domain.Account{}, err
	}
	if tag.RowsAffected() == 0 {
		return domain.Account{}, store.ErrInvalidToken
	}
	event.AggregateID = id
	event.Payload = map[string]any{"accountId": id, "verificationState": "VERIFIED"}
	if err = insertEvent(ctx, tx, event); err != nil {
		return domain.Account{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Account{}, err
	}
	return r.AccountByID(ctx, id)
}

func (r *Repository) CredentialsByEmail(ctx context.Context, email string) (store.Credential, error) {
	var c store.Credential
	var roles []string
	err := r.pool.QueryRow(ctx, `SELECT a.account_id,a.normalized_email,a.status,a.verification_state,a.token_version,a.created_at,a.updated_at,c.password_hash,COALESCE(array_agg(ra.role) FILTER (WHERE ra.revoked_at IS NULL),ARRAY[]::text[]) FROM account a JOIN credential c USING(account_id) LEFT JOIN role_assignment ra USING(account_id) WHERE a.normalized_email=$1 GROUP BY a.account_id,c.password_hash`, email).Scan(&c.Account.ID, &c.Account.Email, &c.Account.Status, &c.Account.Verification, &c.TokenVersion, &c.Account.CreatedAt, &c.Account.UpdatedAt, &c.PasswordHash, &roles)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, store.ErrNotFound
	}
	c.Account.Roles = roles
	return c, err
}
func (r *Repository) AccountByID(ctx context.Context, id string) (domain.Account, error) {
	var a domain.Account
	err := r.pool.QueryRow(ctx, `SELECT a.account_id,a.normalized_email,a.status,a.verification_state,a.token_version,a.created_at,a.updated_at,COALESCE(array_agg(ra.role) FILTER (WHERE ra.revoked_at IS NULL),ARRAY[]::text[]) FROM account a LEFT JOIN role_assignment ra USING(account_id) WHERE a.account_id=$1 GROUP BY a.account_id`, id).Scan(&a.ID, &a.Email, &a.Status, &a.Verification, &a.TokenVersion, &a.CreatedAt, &a.UpdatedAt, &a.Roles)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, store.ErrNotFound
	}
	return a, err
}
func (r *Repository) CreateSession(ctx context.Context, s domain.Session) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO refresh_session(session_id,family_id,account_id,token_hash,expires_at,created_at) VALUES($1,$2,$3,$4,$5,now())`, s.ID, s.FamilyID, s.AccountID, s.TokenHash, s.ExpiresAt)
	return err
}
func (r *Repository) RotateSession(ctx context.Context, oldHash []byte, next domain.Session, now time.Time) (domain.Account, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Account{}, err
	}
	defer tx.Rollback(ctx)
	var id, family, account string
	var expires time.Time
	var revoked *time.Time
	err = tx.QueryRow(ctx, `SELECT session_id,family_id,account_id,expires_at,revoked_at FROM refresh_session WHERE token_hash=$1 FOR UPDATE`, oldHash).Scan(&id, &family, &account, &expires, &revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Account{}, store.ErrInvalidToken
	}
	if err != nil {
		return domain.Account{}, err
	}
	if revoked != nil {
		_, _ = tx.Exec(ctx, `UPDATE refresh_session SET revoked_at=COALESCE(revoked_at,$2),reuse_detected_at=COALESCE(reuse_detected_at,$2) WHERE family_id=$1`, family, now)
		_ = tx.Commit(ctx)
		return domain.Account{}, store.ErrRefreshReuse
	}
	if !expires.After(now) {
		return domain.Account{}, store.ErrInvalidToken
	}
	next.FamilyID = family
	next.AccountID = account
	if _, err = tx.Exec(ctx, `UPDATE refresh_session SET revoked_at=$2,rotated_at=$2,replaced_by=$3 WHERE session_id=$1`, id, now, next.ID); err != nil {
		return domain.Account{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO refresh_session(session_id,family_id,account_id,token_hash,expires_at,created_at) VALUES($1,$2,$3,$4,$5,$6)`, next.ID, next.FamilyID, next.AccountID, next.TokenHash, next.ExpiresAt, now); err != nil {
		return domain.Account{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Account{}, err
	}
	return r.AccountByID(ctx, account)
}
func (r *Repository) RevokeSession(ctx context.Context, hash []byte, now time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE refresh_session SET revoked_at=COALESCE(revoked_at,$2) WHERE token_hash=$1`, hash, now)
	return err
}

func (r *Repository) GetMe(ctx context.Context, id string) (domain.Me, error) {
	a, err := r.AccountByID(ctx, id)
	if err != nil {
		return domain.Me{}, err
	}
	var p domain.Profile
	err = r.pool.QueryRow(ctx, `SELECT account_id,nickname,date_of_birth::text,broad_location,bio,visibility,approval_state,version,updated_at,COALESCE((SELECT array_agg(interest ORDER BY interest) FROM profile_interest WHERE account_id=$1),ARRAY[]::text[]) FROM profile WHERE account_id=$1`, id).Scan(&p.AccountID, &p.Nickname, &p.DateOfBirth, &p.BroadLocation, &p.Bio, &p.Visibility, &p.Approval, &p.Version, &p.UpdatedAt, &p.Interests)
	if err != nil {
		return domain.Me{}, err
	}
	var pref domain.Preferences
	var intentions, interested, languages, deal []byte
	err = r.pool.QueryRow(ctx, `SELECT min_age,max_age,intentions,interested_in,languages,deal_breakers,version,updated_at FROM matching_preference WHERE account_id=$1`, id).Scan(&pref.MinAge, &pref.MaxAge, &intentions, &interested, &languages, &deal, &pref.Version, &pref.UpdatedAt)
	var pp *domain.Preferences
	if err == nil {
		_ = json.Unmarshal(intentions, &pref.Intentions)
		_ = json.Unmarshal(interested, &pref.InterestedIn)
		_ = json.Unmarshal(languages, &pref.Languages)
		_ = json.Unmarshal(deal, &pref.DealBreakers)
		pp = &pref
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Me{}, err
	}
	return domain.Me{Account: a, Profile: p, Preferences: pp}, nil
}

func (r *Repository) UpdateProfile(ctx context.Context, id string, patch domain.ProfilePatch, event domain.Event) (domain.Profile, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Profile{}, err
	}
	defer tx.Rollback(ctx)
	var p domain.Profile
	err = tx.QueryRow(ctx, `SELECT account_id,nickname,date_of_birth::text,broad_location,bio,visibility,approval_state,version,updated_at FROM profile WHERE account_id=$1 FOR UPDATE`, id).Scan(&p.AccountID, &p.Nickname, &p.DateOfBirth, &p.BroadLocation, &p.Bio, &p.Visibility, &p.Approval, &p.Version, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, store.ErrNotFound
	}
	if err != nil {
		return p, err
	}
	if patch.ExpectedVersion > 0 && patch.ExpectedVersion != p.Version {
		return p, store.ErrConflict
	}
	if patch.Nickname != nil {
		p.Nickname = *patch.Nickname
	}
	if patch.DateOfBirth != nil {
		p.DateOfBirth = *patch.DateOfBirth
	}
	if patch.BroadLocation != nil {
		p.BroadLocation = *patch.BroadLocation
	}
	if patch.Bio != nil {
		p.Bio = *patch.Bio
	}
	if patch.Visibility != nil {
		p.Visibility = *patch.Visibility
	}
	p.Version++
	p.UpdatedAt = event.OccurredAt
	_, err = tx.Exec(ctx, `UPDATE profile SET nickname=$2,date_of_birth=$3,broad_location=$4,bio=$5,visibility=$6,version=$7,updated_at=$8 WHERE account_id=$1`, id, p.Nickname, p.DateOfBirth, p.BroadLocation, p.Bio, p.Visibility, p.Version, p.UpdatedAt)
	if err != nil {
		return p, err
	}
	if patch.Interests != nil {
		if _, err = tx.Exec(ctx, `DELETE FROM profile_interest WHERE account_id=$1`, id); err != nil {
			return p, err
		}
		for _, v := range *patch.Interests {
			if _, err = tx.Exec(ctx, `INSERT INTO profile_interest(account_id,interest) VALUES($1,$2)`, id, v); err != nil {
				return p, err
			}
		}
		p.Interests = *patch.Interests
	} else {
		_ = tx.QueryRow(ctx, `SELECT COALESCE(array_agg(interest ORDER BY interest),ARRAY[]::text[]) FROM profile_interest WHERE account_id=$1`, id).Scan(&p.Interests)
	}
	event.AggregateID = id
	event.Payload = map[string]any{"accountId": id, "profileVersion": p.Version, "visibility": p.Visibility, "approvalState": p.Approval}
	if err = insertEvent(ctx, tx, event); err != nil {
		return p, err
	}
	if err = tx.Commit(ctx); err != nil {
		return p, err
	}
	return p, nil
}

func (r *Repository) ReplacePreferences(ctx context.Context, id string, in domain.PreferenceInput) (domain.Preferences, error) {
	intentions, _ := json.Marshal(in.Intentions)
	interested, _ := json.Marshal(in.InterestedIn)
	languages, _ := json.Marshal(in.Languages)
	deal, _ := json.Marshal(in.DealBreakers)
	var p domain.Preferences
	var a, b, c, d []byte
	err := r.pool.QueryRow(ctx, `INSERT INTO matching_preference(account_id,min_age,max_age,intentions,interested_in,languages,deal_breakers,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,now()) ON CONFLICT(account_id) DO UPDATE SET min_age=EXCLUDED.min_age,max_age=EXCLUDED.max_age,intentions=EXCLUDED.intentions,interested_in=EXCLUDED.interested_in,languages=EXCLUDED.languages,deal_breakers=EXCLUDED.deal_breakers,version=matching_preference.version+1,updated_at=now() RETURNING min_age,max_age,intentions,interested_in,languages,deal_breakers,version,updated_at`, id, in.MinAge, in.MaxAge, intentions, interested, languages, deal).Scan(&p.MinAge, &p.MaxAge, &a, &b, &c, &d, &p.Version, &p.UpdatedAt)
	_ = json.Unmarshal(a, &p.Intentions)
	_ = json.Unmarshal(b, &p.InterestedIn)
	_ = json.Unmarshal(c, &p.Languages)
	_ = json.Unmarshal(d, &p.DealBreakers)
	return p, err
}

func scanCommunity(row pgx.Row) (domain.CommunityProfile, string, error) {
	var p domain.CommunityProfile
	var dob string
	err := row.Scan(&p.ProfileID, &p.Nickname, &dob, &p.BroadLocation, &p.Bio, &p.Interests)
	p.AgeBand = domain.AgeBand(dob, time.Now().UTC())
	return p, dob, err
}
func (r *Repository) ListCommunity(ctx context.Context, viewer, cursor string, limit int) ([]domain.CommunityProfile, string, error) {
	rows, err := r.pool.Query(ctx, `SELECT p.account_id,p.nickname,p.date_of_birth::text,p.broad_location,p.bio,COALESCE(array_agg(pi.interest ORDER BY pi.interest) FILTER (WHERE pi.interest IS NOT NULL),ARRAY[]::text[]) FROM profile p JOIN account a ON a.account_id=p.account_id LEFT JOIN profile_interest pi ON pi.account_id=p.account_id WHERE a.status='ACTIVE' AND a.verification_state='VERIFIED' AND p.visibility='COMMUNITY' AND p.approval_state='APPROVED' AND p.account_id<>$1 AND ($2='' OR p.account_id::text>$2) AND NOT EXISTS(SELECT 1 FROM member_block b WHERE b.revoked_at IS NULL AND ((b.blocker_account_id=$1 AND b.blocked_account_id=p.account_id) OR (b.blocker_account_id=p.account_id AND b.blocked_account_id=$1))) GROUP BY p.account_id ORDER BY p.account_id LIMIT $3`, viewer, cursor, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := []domain.CommunityProfile{}
	for rows.Next() {
		p, _, e := scanCommunity(rows)
		if e != nil {
			return nil, "", e
		}
		items = append(items, p)
	}
	next := ""
	if len(items) > limit {
		next = items[limit-1].ProfileID
		items = items[:limit]
	}
	return items, next, rows.Err()
}
func (r *Repository) GetCommunity(ctx context.Context, viewer, id string) (domain.CommunityProfile, error) {
	p, _, err := scanCommunity(r.pool.QueryRow(ctx, `SELECT p.account_id,p.nickname,p.date_of_birth::text,p.broad_location,p.bio,COALESCE(array_agg(pi.interest ORDER BY pi.interest) FILTER (WHERE pi.interest IS NOT NULL),ARRAY[]::text[]) FROM profile p JOIN account a ON a.account_id=p.account_id LEFT JOIN profile_interest pi ON pi.account_id=p.account_id WHERE p.account_id=$2 AND a.status='ACTIVE' AND a.verification_state='VERIFIED' AND p.visibility='COMMUNITY' AND p.approval_state='APPROVED' AND NOT EXISTS(SELECT 1 FROM member_block b WHERE b.revoked_at IS NULL AND ((b.blocker_account_id=$1 AND b.blocked_account_id=p.account_id) OR (b.blocker_account_id=p.account_id AND b.blocked_account_id=$1))) GROUP BY p.account_id`, viewer, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return p, store.ErrNotFound
	}
	return p, err
}

func (r *Repository) Block(ctx context.Context, actor, target string, event domain.Event) error {
	return r.blockChange(ctx, actor, target, false, event)
}
func (r *Repository) Unblock(ctx context.Context, actor, target string, event domain.Event) error {
	return r.blockChange(ctx, actor, target, true, event)
}
func (r *Repository) blockChange(ctx context.Context, actor, target string, revoke bool, event domain.Event) error {
	if actor == target {
		return store.ErrConflict
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if revoke {
		tag, e := tx.Exec(ctx, `UPDATE member_block SET revoked_at=$3 WHERE blocker_account_id=$1 AND blocked_account_id=$2 AND revoked_at IS NULL`, actor, target, event.OccurredAt)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return store.ErrNotFound
		}
	} else {
		_, _ = tx.Exec(ctx, `UPDATE member_block SET revoked_at=$3 WHERE blocker_account_id=$1 AND blocked_account_id=$2 AND revoked_at IS NULL`, actor, target, event.OccurredAt)
		if _, err = tx.Exec(ctx, `INSERT INTO member_block(blocker_account_id,blocked_account_id,created_at) VALUES($1,$2,$3)`, actor, target, event.OccurredAt); err != nil {
			return err
		}
	}
	event.AggregateID = actor
	event.Payload = map[string]any{"blockerAccountId": actor, "blockedAccountId": target}
	if err = insertEvent(ctx, tx, event); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) Deactivate(ctx context.Context, id string, event domain.Event) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE account SET status='DEACTIVATED',token_version=token_version+1,deactivated_at=$2,updated_at=$2 WHERE account_id=$1 AND status<>'DEACTIVATED'`, id, event.OccurredAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return store.ErrConflict
	}
	_, _ = tx.Exec(ctx, `UPDATE refresh_session SET revoked_at=COALESCE(revoked_at,$2) WHERE account_id=$1`, id, event.OccurredAt)
	_, _ = tx.Exec(ctx, `UPDATE profile SET visibility='HIDDEN',version=version+1,updated_at=$2 WHERE account_id=$1`, id, event.OccurredAt)
	event.AggregateID = id
	event.Payload = map[string]any{"accountId": id}
	if err = insertEvent(ctx, tx, event); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) SetProfileDecision(ctx context.Context, actor, target, decision, reason string, event domain.Event) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	visibility := "COMMUNITY"
	if decision == domain.ApprovalHidden {
		visibility = domain.VisibilityHidden
	}
	tag, err := tx.Exec(ctx, `UPDATE profile SET approval_state=$2,visibility=$3,version=version+1,updated_at=$4 WHERE account_id=$1`, target, decision, visibility, event.OccurredAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_log(audit_id,actor_id,target_id,action,reason,correlation_id,occurred_at,new_state) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, uuid.NewString(), actor, target, "PROFILE_"+decision, reason, event.CorrelationID, event.OccurredAt, fmt.Sprintf(`{"approvalState":%q,"visibility":%q}`, decision, visibility))
	if err != nil {
		return err
	}
	event.AggregateID = target
	event.Payload = map[string]any{"accountId": target, "approvalState": decision, "visibility": visibility}
	if err = insertEvent(ctx, tx, event); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) ClaimOutbox(ctx context.Context, limit int) ([]store.OutboxRecord, error) {
	rows, err := r.pool.Query(ctx, `WITH picked AS (SELECT event_id FROM outbox WHERE published_at IS NULL AND (claimed_at IS NULL OR claimed_at<now()-interval '1 minute') ORDER BY occurred_at LIMIT $1 FOR UPDATE SKIP LOCKED) UPDATE outbox o SET claimed_at=now(),attempts=attempts+1 FROM picked WHERE o.event_id=picked.event_id RETURNING o.event_id,o.routing_key,jsonb_build_object('eventId',o.event_id,'eventType',o.event_type,'schemaVersion',o.schema_version,'occurredAt',o.occurred_at,'aggregateId',o.aggregate_id,'correlationId',o.correlation_id,'causationId',o.causation_id,'actorId',o.actor_id,'payload',o.payload)::text`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []store.OutboxRecord{}
	for rows.Next() {
		var v store.OutboxRecord
		var body string
		if err = rows.Scan(&v.ID, &v.RoutingKey, &body); err != nil {
			return nil, err
		}
		v.Body = []byte(body)
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *Repository) MarkOutboxPublished(ctx context.Context, id string, at time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE outbox SET published_at=$2,claimed_at=NULL WHERE event_id=$1`, id, at)
	return err
}

func insertEvent(ctx context.Context, tx pgx.Tx, e domain.Event) error {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox(event_id,event_type,schema_version,occurred_at,aggregate_id,correlation_id,causation_id,actor_id,routing_key,payload) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, e.EventID, e.EventType, e.SchemaVersion, e.OccurredAt, e.AggregateID, e.CorrelationID, e.CausationID, e.ActorID, "account."+e.EventType, payload)
	return err
}
func isUnique(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}
