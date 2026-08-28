package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/domain"
	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/engine"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Store                { return &Store{db: db} }
func (s *Store) Ready(ctx context.Context) error { return s.db.Ping(ctx) }

func (s *Store) Generate(ctx context.Context, p domain.Principal, eventID, key, correlation string) (domain.Run, error) {
	if key == "" {
		return domain.Run{}, problem(400, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key is required")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Run{}, err
	}
	defer tx.Rollback(ctx)
	if err = s.authorizeEvent(ctx, tx, p, eventID); err != nil {
		return domain.Run{}, err
	}
	var existing string
	if err = tx.QueryRow(ctx, `SELECT run_id::text FROM matching_run WHERE event_id=$1 AND idempotency_key=$2`, eventID, key).Scan(&existing); err == nil {
		return s.getTx(ctx, tx, existing)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Run{}, err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, eventID); err != nil {
		return domain.Run{}, err
	}
	var rulesVersion, status string
	if err = tx.QueryRow(ctx, `SELECT ruleset_version,event_status FROM event_scope WHERE event_id=$1`, eventID).Scan(&rulesVersion, &status); err != nil {
		return domain.Run{}, err
	}
	if status != "MATCHING" {
		return domain.Run{}, problem(409, "EVENT_NOT_READY_FOR_MATCHING", "Event fixture is not in matching state")
	}
	var rawRules []byte
	if err = tx.QueryRow(ctx, `SELECT configuration FROM ruleset WHERE version=$1 AND status='APPROVED'`, rulesVersion).Scan(&rawRules); err != nil {
		return domain.Run{}, err
	}
	var rules domain.Ruleset
	if err = json.Unmarshal(rawRules, &rules); err != nil {
		return domain.Run{}, err
	}
	rows, err := tx.Query(ctx, `SELECT input,source_version FROM participant_projection WHERE event_id=$1 ORDER BY account_id`, eventID)
	if err != nil {
		return domain.Run{}, err
	}
	defer rows.Close()
	people := []domain.Participant{}
	versions := map[string]int64{}
	for rows.Next() {
		var raw []byte
		var source int64
		if err = rows.Scan(&raw, &source); err != nil {
			return domain.Run{}, err
		}
		var person domain.Participant
		if err = json.Unmarshal(raw, &person); err != nil {
			return domain.Run{}, err
		}
		people = append(people, person)
		versions[person.AccountID] = source
	}
	if err = rows.Err(); err != nil {
		return domain.Run{}, err
	}
	if len(people) < 2 {
		return domain.Run{}, problem(422, "INSUFFICIENT_PARTICIPANTS", "At least two fixture participants are required")
	}
	generated, err := engine.Generate(people, rules)
	if err != nil {
		return domain.Run{}, problem(422, "MATCHING_INPUT_INVALID", err.Error())
	}
	runID := uuid.NewString()
	var runVersion int
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(run_version),0)+1 FROM matching_run WHERE event_id=$1`, eventID).Scan(&runVersion); err != nil {
		return domain.Run{}, err
	}
	now := time.Now().UTC()
	eligible := 0
	for _, candidate := range generated.Candidates {
		if candidate.Eligible {
			eligible++
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO matching_run(run_id,event_id,run_version,status,ruleset_version,optimizer_version,tie_break_policy,idempotency_key,created_by,participant_count,eligible_pair_count,created_at,updated_at) VALUES($1,$2,$3,'GENERATED',$4,$5,'LEXICOGRAPHIC_ACCOUNT_ID',$6,$7,$8,$9,$10,$10)`, runID, eventID, runVersion, rules.Version, engine.OptimizerVersion, key, p.Subject, len(people), eligible, now); err != nil {
		return domain.Run{}, err
	}
	for _, person := range people {
		raw, _ := json.Marshal(person)
		if _, err = tx.Exec(ctx, `INSERT INTO participant_snapshot(snapshot_id,run_id,account_id,participant_code,group_code,source_version,input,snapshot_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, uuid.NewString(), runID, person.AccountID, person.ParticipantCode, person.Group, versions[person.AccountID], raw, now); err != nil {
			return domain.Run{}, err
		}
	}
	for _, candidate := range generated.Candidates {
		rejections, _ := json.Marshal(candidate.RejectionCodes)
		components, _ := json.Marshal(candidate.Components)
		reasons, _ := json.Marshal(candidate.SafeReasons)
		var total any
		if candidate.Eligible {
			total = candidate.TotalScore
		}
		if _, err = tx.Exec(ctx, `INSERT INTO candidate(candidate_id,run_id,participant_a,participant_b,canonical_pair_key,eligible,rejection_codes,components,total_score,safe_reasons) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, uuid.NewString(), runID, candidate.ParticipantA, candidate.ParticipantB, pairKey(candidate.ParticipantA, candidate.ParticipantB), candidate.Eligible, rejections, components, total, reasons); err != nil {
			return domain.Run{}, err
		}
	}
	for index, pair := range generated.Pairings {
		reasons, _ := json.Marshal(pair.SafeReasons)
		suggestionID := uuid.NewString()
		if _, err = tx.Exec(ctx, `INSERT INTO pairing_suggestion(suggestion_id,run_id,participant_a,participant_b,score,safe_reasons,optimizer_order) VALUES($1,$2,$3,$4,$5,$6,$7)`, suggestionID, runID, pair.ParticipantA, pair.ParticipantB, pair.Score, reasons, index+1); err != nil {
			return domain.Run{}, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO pairing_selection(selection_id,run_id,participant_a,participant_b,score,safe_reasons,source) VALUES($1,$2,$3,$4,$5,$6,'ALGORITHM')`, suggestionID, runID, pair.ParticipantA, pair.ParticipantB, pair.Score, reasons); err != nil {
			return domain.Run{}, err
		}
	}
	for _, item := range generated.Unmatched {
		if _, err = tx.Exec(ctx, `INSERT INTO unmatched_participant(run_id,account_id,reason_code) VALUES($1,$2,$3)`, runID, item.ParticipantID, item.Reason); err != nil {
			return domain.Run{}, err
		}
	}
	if err = s.fact(ctx, tx, "MatchingRunGenerated", runID, p.Subject, correlation, map[string]any{"runId": runID, "eventId": eventID, "runVersion": runVersion, "rulesetVersion": rules.Version, "participantCount": len(people), "pairingCount": len(generated.Pairings), "unmatchedCount": len(generated.Unmatched)}); err != nil {
		return domain.Run{}, err
	}
	if err = s.audit(ctx, tx, p.Subject, runID, "MATCHING_RUN_GENERATED", "", map[string]any{}, map[string]any{"status": "GENERATED"}, correlation); err != nil {
		return domain.Run{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Run{}, err
	}
	return s.Get(ctx, p, runID)
}

func (s *Store) List(ctx context.Context, p domain.Principal, eventID string) ([]domain.Run, error) {
	if err := s.authorizeEvent(ctx, s.db, p, eventID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, runSelect+` WHERE event_id=$1 ORDER BY run_version DESC`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Run{}
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}
func (s *Store) Get(ctx context.Context, p domain.Principal, runID string) (domain.Run, error) {
	run, err := s.getTx(ctx, s.db, runID)
	if err != nil {
		return run, err
	}
	if err = s.authorizeEvent(ctx, s.db, p, run.EventID); err != nil {
		return domain.Run{}, err
	}
	return run, nil
}

func (s *Store) Review(ctx context.Context, p domain.Principal, runID string, expected int64, correlation string) (domain.Run, error) {
	return s.transition(ctx, p, runID, expected, "GENERATED", "UNDER_REVIEW", "MatchingRunReviewStarted", correlation)
}
func (s *Store) Publish(ctx context.Context, p domain.Principal, runID string, expected int64, correlation string) (domain.Run, error) {
	return s.transition(ctx, p, runID, expected, "LOCKED", "PUBLISHED", "PairingsPublished", correlation)
}
func (s *Store) transition(ctx context.Context, p domain.Principal, runID string, expected int64, from, to, eventType, correlation string) (domain.Run, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Run{}, err
	}
	defer tx.Rollback(ctx)
	var eventID, status string
	var version int64
	if err = tx.QueryRow(ctx, `SELECT event_id::text,status,aggregate_version FROM matching_run WHERE run_id=$1 FOR UPDATE`, runID).Scan(&eventID, &status, &version); err != nil {
		return domain.Run{}, notFound(err)
	}
	if err = s.authorizeEvent(ctx, tx, p, eventID); err != nil {
		return domain.Run{}, err
	}
	if status == to {
		return s.getTx(ctx, tx, runID)
	}
	if status != from || version != expected {
		return domain.Run{}, problem(409, "MATCHING_RUN_STATE_CONFLICT", "Run state or expectedVersion is stale")
	}
	tag, err := tx.Exec(ctx, `UPDATE matching_run SET status=$3,aggregate_version=aggregate_version+1,updated_at=now(),published_at=CASE WHEN $3='PUBLISHED' THEN now() ELSE published_at END WHERE run_id=$1 AND aggregate_version=$2 AND status=$4`, runID, expected, to, from)
	if err != nil {
		return domain.Run{}, err
	}
	if tag.RowsAffected() != 1 {
		return domain.Run{}, problem(409, "MATCHING_RUN_STATE_CONFLICT", "Run state or expectedVersion is stale")
	}
	if err = s.fact(ctx, tx, eventType, runID, p.Subject, correlation, map[string]any{"runId": runID, "eventId": eventID, "status": to}); err != nil {
		return domain.Run{}, err
	}
	if err = s.audit(ctx, tx, p.Subject, runID, "MATCHING_RUN_"+to, "", map[string]any{"status": from}, map[string]any{"status": to}, correlation); err != nil {
		return domain.Run{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Run{}, err
	}
	return s.Get(ctx, p, runID)
}

func (s *Store) Override(ctx context.Context, p domain.Principal, runID, removeID, a, b, reason, key, correlation string, expected int64) (domain.Run, error) {
	if strings.TrimSpace(reason) == "" || key == "" {
		return domain.Run{}, problem(400, "OVERRIDE_INPUT_REQUIRED", "Reason and Idempotency-Key are required")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Run{}, err
	}
	defer tx.Rollback(ctx)
	var eventID, status string
	var version int64
	if err = tx.QueryRow(ctx, `SELECT event_id::text,status,aggregate_version FROM matching_run WHERE run_id=$1 FOR UPDATE`, runID).Scan(&eventID, &status, &version); err != nil {
		return domain.Run{}, notFound(err)
	}
	if err = s.authorizeEvent(ctx, tx, p, eventID); err != nil {
		return domain.Run{}, err
	}
	if status != "UNDER_REVIEW" || version != expected {
		return domain.Run{}, problem(409, "MATCHING_RUN_STATE_CONFLICT", "Run must be under review and expectedVersion current")
	}
	var existing string
	if err = tx.QueryRow(ctx, `SELECT override_id::text FROM pairing_override WHERE run_id=$1 AND idempotency_key=$2`, runID, key).Scan(&existing); err == nil {
		return s.getTx(ctx, tx, runID)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Run{}, err
	}
	var score int
	var reasons []byte
	if err = tx.QueryRow(ctx, `SELECT total_score,safe_reasons FROM candidate WHERE run_id=$1 AND canonical_pair_key=$2 AND eligible=true AND total_score >= (SELECT (configuration->>'minimumScore')::int FROM ruleset r JOIN matching_run mr ON mr.ruleset_version=r.version WHERE mr.run_id=$1)`, runID, pairKey(a, b)).Scan(&score, &reasons); err != nil {
		return domain.Run{}, problem(422, "OVERRIDE_PAIR_INELIGIBLE", "Replacement pair is not eligible or is below the threshold")
	}
	if removeID != "" {
		tag, err := tx.Exec(ctx, `DELETE FROM pairing_selection WHERE run_id=$1 AND selection_id=$2`, runID, removeID)
		if err != nil {
			return domain.Run{}, err
		}
		if tag.RowsAffected() != 1 {
			return domain.Run{}, problem(404, "SELECTION_NOT_FOUND", "Selection to replace was not found")
		}
	}
	var conflicts int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM pairing_selection WHERE run_id=$1 AND (participant_a=$2 OR participant_b=$2 OR participant_a=$3 OR participant_b=$3)`, runID, a, b).Scan(&conflicts); err != nil {
		return domain.Run{}, err
	}
	if conflicts > 0 {
		return domain.Run{}, problem(409, "PARTICIPANT_ALREADY_SELECTED", "A replacement participant is already selected")
	}
	selectionID, overrideID := uuid.NewString(), uuid.NewString()
	if _, err = tx.Exec(ctx, `INSERT INTO pairing_selection(selection_id,run_id,participant_a,participant_b,score,safe_reasons,source) VALUES($1,$2,$3,$4,$5,$6,'OVERRIDE')`, selectionID, runID, a, b, score, reasons); err != nil {
		return domain.Run{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO pairing_override(override_id,run_id,removed_selection_id,replacement_selection_id,actor_id,reason,correlation_id,idempotency_key,created_at) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,now())`, overrideID, runID, removeID, selectionID, p.Subject, reason, correlation, key); err != nil {
		return domain.Run{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE matching_run SET aggregate_version=aggregate_version+1,updated_at=now() WHERE run_id=$1`, runID); err != nil {
		return domain.Run{}, err
	}
	if err = s.audit(ctx, tx, p.Subject, runID, "PAIRING_OVERRIDDEN", reason, map[string]any{"removedSelectionId": removeID}, map[string]any{"participantA": a, "participantB": b}, correlation); err != nil {
		return domain.Run{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Run{}, err
	}
	return s.Get(ctx, p, runID)
}

func (s *Store) Lock(ctx context.Context, p domain.Principal, runID string, expected int64, key, correlation string) (domain.Run, error) {
	if key == "" {
		return domain.Run{}, problem(400, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key is required")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Run{}, err
	}
	defer tx.Rollback(ctx)
	var eventID, status, rulesVersion string
	var version int64
	if err = tx.QueryRow(ctx, `SELECT event_id::text,status,aggregate_version,ruleset_version FROM matching_run WHERE run_id=$1 FOR UPDATE`, runID).Scan(&eventID, &status, &version, &rulesVersion); err != nil {
		return domain.Run{}, notFound(err)
	}
	if err = s.authorizeEvent(ctx, tx, p, eventID); err != nil {
		return domain.Run{}, err
	}
	if status == "LOCKED" || status == "PUBLISHED" {
		return s.getTx(ctx, tx, runID)
	}
	if status != "UNDER_REVIEW" || version != expected {
		return domain.Run{}, problem(409, "MATCHING_RUN_STATE_CONFLICT", "Run must be under review and expectedVersion current")
	}
	var rawRules []byte
	if err = tx.QueryRow(ctx, `SELECT configuration FROM ruleset WHERE version=$1`, rulesVersion).Scan(&rawRules); err != nil {
		return domain.Run{}, err
	}
	var rules domain.Ruleset
	if err = json.Unmarshal(rawRules, &rules); err != nil {
		return domain.Run{}, err
	}
	rows, err := tx.Query(ctx, `SELECT selection_id::text,participant_a::text,participant_b::text,score,safe_reasons FROM pairing_selection WHERE run_id=$1 ORDER BY participant_a`, runID)
	if err != nil {
		return domain.Run{}, err
	}
	type selection struct {
		id, a, b string
		score    int
		reasons  []byte
	}
	selected := []selection{}
	for rows.Next() {
		var item selection
		if err = rows.Scan(&item.id, &item.a, &item.b, &item.score, &item.reasons); err != nil {
			return domain.Run{}, err
		}
		selected = append(selected, item)
	}
	rows.Close()
	if len(selected) == 0 {
		return domain.Run{}, problem(422, "NO_PAIRINGS_SELECTED", "At least one pairing must be selected")
	}
	now := time.Now().UTC()
	pairingIDs := []string{}
	for _, item := range selected {
		var rawA, rawB []byte
		if err = tx.QueryRow(ctx, `SELECT input FROM participant_projection WHERE event_id=$1 AND account_id=$2`, eventID, item.a).Scan(&rawA); err != nil {
			return domain.Run{}, problem(409, "ELIGIBILITY_CHANGED", "Participant projection is unavailable")
		}
		if err = tx.QueryRow(ctx, `SELECT input FROM participant_projection WHERE event_id=$1 AND account_id=$2`, eventID, item.b).Scan(&rawB); err != nil {
			return domain.Run{}, problem(409, "ELIGIBILITY_CHANGED", "Participant projection is unavailable")
		}
		var a, b domain.Participant
		_ = json.Unmarshal(rawA, &a)
		_ = json.Unmarshal(rawB, &b)
		if !engine.Evaluate(a, b, rules).Eligible {
			return domain.Run{}, problem(409, "ELIGIBILITY_CHANGED", "A selected pair no longer passes hard eligibility rules")
		}
		pairingID := uuid.NewString()
		pairingIDs = append(pairingIDs, pairingID)
		if _, err = tx.Exec(ctx, `INSERT INTO locked_pairing(pairing_id,run_id,event_id,participant_a,participant_b,score,safe_reasons,locked_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, pairingID, runID, eventID, item.a, item.b, item.score, item.reasons, now); err != nil {
			return domain.Run{}, err
		}
		for _, accountID := range []string{item.a, item.b} {
			if _, err = tx.Exec(ctx, `INSERT INTO locked_participant(pairing_id,event_id,round_no,account_id) VALUES($1,$2,1,$3)`, pairingID, eventID, accountID); err != nil {
				return domain.Run{}, problem(409, "PARTICIPANT_LOCK_CONFLICT", "A participant is already locked for this event round")
			}
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE matching_run SET status='LOCKED',aggregate_version=aggregate_version+1,updated_at=$2,locked_at=$2 WHERE run_id=$1`, runID, now); err != nil {
		return domain.Run{}, err
	}
	if err = s.fact(ctx, tx, "PairingsLocked", runID, p.Subject, correlation, map[string]any{"runId": runID, "eventId": eventID, "pairingIds": pairingIDs, "lockedAt": now}); err != nil {
		return domain.Run{}, err
	}
	if err = s.audit(ctx, tx, p.Subject, runID, "PAIRINGS_LOCKED", "", map[string]any{"status": "UNDER_REVIEW"}, map[string]any{"status": "LOCKED", "pairingCount": len(pairingIDs)}, correlation); err != nil {
		return domain.Run{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Run{}, err
	}
	return s.Get(ctx, p, runID)
}

type MemberMatch struct {
	MatchID             string   `json:"matchId"`
	EventID             string   `json:"eventId"`
	PartnerCode         string   `json:"partnerCode"`
	Score               int      `json:"compatibilityScore"`
	SafeReasons         []string `json:"safeReasons"`
	MyResponse          string   `json:"myResponse,omitempty"`
	MutualInterest      bool     `json:"mutualInterest"`
	MyRevealConsent     string   `json:"myRevealConsent,omitempty"`
	BothRevealConsented bool     `json:"bothRevealConsented"`
}

func (s *Store) Mine(ctx context.Context, p domain.Principal) ([]MemberMatch, error) {
	rows, err := s.db.Query(ctx, `SELECT lp.pairing_id::text,lp.event_id::text,CASE WHEN lp.participant_a=$1 THEN sb.participant_code ELSE sa.participant_code END,lp.score,lp.safe_reasons,COALESCE(mine.response,''),(SELECT count(*)=2 FROM match_response mr WHERE mr.pairing_id=lp.pairing_id AND mr.response='INTERESTED'),COALESCE(rc.decision,''),(SELECT count(*)=2 FROM reveal_consent x WHERE x.pairing_id=lp.pairing_id AND x.decision='GRANTED') FROM locked_pairing lp JOIN matching_run r ON r.run_id=lp.run_id AND r.status='PUBLISHED' JOIN participant_snapshot sa ON sa.run_id=r.run_id AND sa.account_id=lp.participant_a JOIN participant_snapshot sb ON sb.run_id=r.run_id AND sb.account_id=lp.participant_b LEFT JOIN match_response mine ON mine.pairing_id=lp.pairing_id AND mine.account_id=$1 LEFT JOIN reveal_consent rc ON rc.pairing_id=lp.pairing_id AND rc.account_id=$1 WHERE lp.participant_a=$1 OR lp.participant_b=$1 ORDER BY lp.locked_at DESC`, p.Subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MemberMatch{}
	for rows.Next() {
		var item MemberMatch
		var reasons []byte
		if err = rows.Scan(&item.MatchID, &item.EventID, &item.PartnerCode, &item.Score, &reasons, &item.MyResponse, &item.MutualInterest, &item.MyRevealConsent, &item.BothRevealConsented); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(reasons, &item.SafeReasons)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) Respond(ctx context.Context, p domain.Principal, matchID, response, questionVersion, key, correlation string) error {
	if response != "INTERESTED" && response != "PASS" {
		return problem(422, "MATCH_RESPONSE_INVALID", "Response must be INTERESTED or PASS")
	}
	return s.memberCommand(ctx, p, matchID, key, func(ctx context.Context, tx pgx.Tx, eventID string) error {
		var existingResponse, existingVersion string
		err := tx.QueryRow(ctx, `SELECT response,question_version FROM match_response WHERE pairing_id=$1 AND account_id=$2`, matchID, p.Subject).Scan(&existingResponse, &existingVersion)
		if err == nil {
			if existingResponse == response && existingVersion == questionVersion {
				return nil
			}
			return problem(409, "MATCH_RESPONSE_ALREADY_RECORDED", "A different response is already recorded for this match")
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO match_response(pairing_id,account_id,response,question_version,idempotency_key,recorded_at) VALUES($1,$2,$3,$4,$5,now()) ON CONFLICT(idempotency_key) DO NOTHING`, matchID, p.Subject, response, questionVersion, key)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return problem(409, "IDEMPOTENCY_KEY_REUSED", "Idempotency-Key was already used for a different command")
		}
		if err = s.fact(ctx, tx, "MatchResponseRecorded", matchID, p.Subject, correlation, map[string]any{"matchId": matchID, "eventId": eventID, "accountId": p.Subject, "response": response, "questionVersion": questionVersion}); err != nil {
			return err
		}
		var mutual bool
		_ = tx.QueryRow(ctx, `SELECT count(*)=2 FROM match_response WHERE pairing_id=$1 AND response='INTERESTED'`, matchID).Scan(&mutual)
		if mutual {
			return s.fact(ctx, tx, "MutualInterestEstablished", matchID, p.Subject, correlation, map[string]any{"matchId": matchID, "eventId": eventID, "policyVersion": questionVersion})
		}
		return nil
	})
}
func (s *Store) Consent(ctx context.Context, p domain.Principal, matchID, decision, policyVersion, key, correlation string) error {
	if decision != "GRANTED" && decision != "REVOKED" {
		return problem(422, "REVEAL_CONSENT_INVALID", "Decision must be GRANTED or REVOKED")
	}
	return s.memberCommand(ctx, p, matchID, key, func(ctx context.Context, tx pgx.Tx, eventID string) error {
		var mutual bool
		if err := tx.QueryRow(ctx, `SELECT count(*)=2 FROM match_response WHERE pairing_id=$1 AND response='INTERESTED'`, matchID).Scan(&mutual); err != nil {
			return err
		}
		if !mutual {
			return problem(409, "MUTUAL_INTEREST_REQUIRED", "Reveal consent is available only after mutual interest")
		}
		var recordedMatch, recordedAccount, recordedDecision, recordedPolicy string
		err := tx.QueryRow(ctx, `SELECT pairing_id::text,account_id::text,decision,policy_version FROM reveal_consent_history WHERE idempotency_key=$1`, key).Scan(&recordedMatch, &recordedAccount, &recordedDecision, &recordedPolicy)
		if err == nil {
			if recordedMatch == matchID && recordedAccount == p.Subject && recordedDecision == decision && recordedPolicy == policyVersion {
				return nil
			}
			return problem(409, "IDEMPOTENCY_KEY_REUSED", "Idempotency-Key was already used for a different consent decision")
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var priorDecision, priorPolicy, priorKey string
		err = tx.QueryRow(ctx, `SELECT decision,policy_version,idempotency_key FROM reveal_consent WHERE pairing_id=$1 AND account_id=$2`, matchID, p.Subject).Scan(&priorDecision, &priorPolicy, &priorKey)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var nextVersion int64
		if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(decision_version),0)+1 FROM reveal_consent_history WHERE pairing_id=$1 AND account_id=$2`, matchID, p.Subject).Scan(&nextVersion); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO reveal_consent_history(pairing_id,account_id,decision_version,decision,policy_version,idempotency_key,recorded_at) VALUES($1,$2,$3,$4,$5,$6,now()) ON CONFLICT(idempotency_key) DO NOTHING`, matchID, p.Subject, nextVersion, decision, policyVersion, key)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return problem(409, "IDEMPOTENCY_KEY_REUSED", "Idempotency-Key was already used for a different consent decision")
		}
		if priorDecision == decision && priorPolicy == policyVersion {
			return nil
		}
		if _, err = tx.Exec(ctx, `INSERT INTO reveal_consent(pairing_id,account_id,decision,policy_version,idempotency_key,recorded_at) VALUES($1,$2,$3,$4,$5,now()) ON CONFLICT(pairing_id,account_id) DO UPDATE SET decision=EXCLUDED.decision,policy_version=EXCLUDED.policy_version,idempotency_key=EXCLUDED.idempotency_key,recorded_at=EXCLUDED.recorded_at`, matchID, p.Subject, decision, policyVersion, key); err != nil {
			return err
		}
		eventType := "RevealConsentGranted"
		if decision == "REVOKED" {
			eventType = "RevealConsentRevoked"
		}
		if err = s.fact(ctx, tx, eventType, matchID, p.Subject, correlation, map[string]any{"matchId": matchID, "eventId": eventID, "accountId": p.Subject, "policyVersion": policyVersion, "decisionVersion": nextVersion}); err != nil {
			return err
		}
		return s.audit(ctx, tx, p.Subject, matchID, "REVEAL_CONSENT_CHANGED", "", map[string]any{"decision": priorDecision, "policyVersion": priorPolicy}, map[string]any{"decision": decision, "policyVersion": policyVersion, "decisionVersion": nextVersion}, correlation)
	})
}
func (s *Store) Feedback(ctx context.Context, p domain.Principal, matchID string, comfort, quality int, safety bool, key, correlation string) error {
	if comfort < 1 || comfort > 5 || quality < 1 || quality > 5 {
		return problem(422, "FEEDBACK_INVALID", "Ratings must be between 1 and 5")
	}
	return s.memberCommand(ctx, p, matchID, key, func(ctx context.Context, tx pgx.Tx, eventID string) error {
		var existingComfort, existingQuality int
		var existingSafety bool
		err := tx.QueryRow(ctx, `SELECT comfort_rating,quality_rating,safety_concern FROM match_feedback WHERE pairing_id=$1 AND account_id=$2`, matchID, p.Subject).Scan(&existingComfort, &existingQuality, &existingSafety)
		if err == nil {
			if existingComfort == comfort && existingQuality == quality && existingSafety == safety {
				return nil
			}
			return problem(409, "MATCH_FEEDBACK_ALREADY_SUBMITTED", "Different feedback is already recorded for this match")
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		feedbackID := uuid.NewString()
		tag, err := tx.Exec(ctx, `INSERT INTO match_feedback(feedback_id,pairing_id,account_id,comfort_rating,quality_rating,safety_concern,idempotency_key,submitted_at) VALUES($1,$2,$3,$4,$5,$6,$7,now()) ON CONFLICT(idempotency_key) DO NOTHING`, feedbackID, matchID, p.Subject, comfort, quality, safety, key)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return problem(409, "IDEMPOTENCY_KEY_REUSED", "Idempotency-Key was already used for a different command")
		}
		return s.fact(ctx, tx, "MatchFeedbackSubmitted", matchID, p.Subject, correlation, map[string]any{"feedbackId": feedbackID, "matchId": matchID, "eventId": eventID, "accountId": p.Subject, "safetyConcern": safety})
	})
}
func (s *Store) memberCommand(ctx context.Context, p domain.Principal, matchID, key string, command func(context.Context, pgx.Tx, string) error) error {
	if key == "" {
		return problem(400, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key is required")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, matchID+"|"+p.Subject); err != nil {
		return err
	}
	var eventID string
	var published bool
	if err = tx.QueryRow(ctx, `SELECT lp.event_id::text,r.status='PUBLISHED' FROM locked_pairing lp JOIN matching_run r ON r.run_id=lp.run_id WHERE lp.pairing_id=$1 AND (lp.participant_a=$2 OR lp.participant_b=$2)`, matchID, p.Subject).Scan(&eventID, &published); err != nil {
		return problem(404, "MATCH_NOT_FOUND", "Published match was not found")
	}
	if !published {
		return problem(409, "MATCH_NOT_PUBLISHED", "Match is not published")
	}
	if err = command(ctx, tx, eventID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

const runSelect = `SELECT run_id::text,event_id::text,run_version,aggregate_version,status,ruleset_version,optimizer_version,tie_break_policy,participant_count,eligible_pair_count,created_by,created_at,updated_at FROM matching_run`

type scanner interface{ Scan(...any) error }

func scanRun(row scanner) (domain.Run, error) {
	var r domain.Run
	err := row.Scan(&r.RunID, &r.EventID, &r.RunVersion, &r.Version, &r.Status, &r.RulesetVersion, &r.OptimizerVersion, &r.TieBreakPolicy, &r.ParticipantCount, &r.EligiblePairCount, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (s *Store) getTx(ctx context.Context, q queryer, runID string) (domain.Run, error) {
	run, err := scanRun(q.QueryRow(ctx, runSelect+` WHERE run_id=$1`, runID))
	if err != nil {
		return run, notFound(err)
	}
	run.Suggestions, err = s.pairings(ctx, q, `SELECT ps.suggestion_id::text,ps.participant_a::text,ps.participant_b::text,a.participant_code,b.participant_code,ps.score,ps.safe_reasons,'ALGORITHM' FROM pairing_suggestion ps JOIN participant_snapshot a ON a.run_id=ps.run_id AND a.account_id=ps.participant_a JOIN participant_snapshot b ON b.run_id=ps.run_id AND b.account_id=ps.participant_b WHERE ps.run_id=$1 ORDER BY ps.optimizer_order`, runID)
	if err != nil {
		return run, err
	}
	run.Selections, err = s.pairings(ctx, q, `SELECT ps.selection_id::text,ps.participant_a::text,ps.participant_b::text,a.participant_code,b.participant_code,ps.score,ps.safe_reasons,ps.source FROM pairing_selection ps JOIN participant_snapshot a ON a.run_id=ps.run_id AND a.account_id=ps.participant_a JOIN participant_snapshot b ON b.run_id=ps.run_id AND b.account_id=ps.participant_b WHERE ps.run_id=$1 ORDER BY ps.participant_a`, runID)
	if err != nil {
		return run, err
	}
	rows, err := q.Query(ctx, `SELECT u.account_id::text,p.participant_code,u.reason_code FROM unmatched_participant u JOIN participant_snapshot p ON p.run_id=u.run_id AND p.account_id=u.account_id WHERE u.run_id=$1 ORDER BY p.participant_code`, runID)
	if err != nil {
		return run, err
	}
	for rows.Next() {
		var item domain.Unmatched
		if err = rows.Scan(&item.ParticipantID, &item.Code, &item.Reason); err != nil {
			rows.Close()
			return run, err
		}
		run.Unmatched = append(run.Unmatched, item)
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT participant_a::text,participant_b::text,eligible,rejection_codes,components,total_score,safe_reasons FROM candidate WHERE run_id=$1 ORDER BY canonical_pair_key`, runID)
	if err != nil {
		return run, err
	}
	for rows.Next() {
		var item domain.Candidate
		var rejection, components, reasons []byte
		var total *int
		if err = rows.Scan(&item.ParticipantA, &item.ParticipantB, &item.Eligible, &rejection, &components, &total, &reasons); err != nil {
			rows.Close()
			return run, err
		}
		_ = json.Unmarshal(rejection, &item.RejectionCodes)
		_ = json.Unmarshal(components, &item.Components)
		_ = json.Unmarshal(reasons, &item.SafeReasons)
		if total != nil {
			item.TotalScore = *total
		}
		run.Candidates = append(run.Candidates, item)
	}
	rows.Close()
	return run, nil
}
func (s *Store) pairings(ctx context.Context, q queryer, sql, runID string) ([]domain.Pairing, error) {
	rows, err := q.Query(ctx, sql, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Pairing{}
	for rows.Next() {
		var item domain.Pairing
		var reasons []byte
		if err = rows.Scan(&item.PairingID, &item.ParticipantA, &item.ParticipantB, &item.ParticipantACode, &item.ParticipantBCode, &item.Score, &reasons, &item.Source); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(reasons, &item.SafeReasons)
		out = append(out, item)
	}
	return out, rows.Err()
}
func (s *Store) authorizeEvent(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, p domain.Principal, eventID string) error {
	if p.HasRole("admin") {
		return nil
	}
	if !p.HasRole("organizer") {
		return problem(403, "MATCHMAKING_SCOPE_REQUIRED", "Organizer or administrator role is required")
	}
	var organizer string
	if err := q.QueryRow(ctx, `SELECT organizer_id FROM event_scope WHERE event_id=$1`, eventID).Scan(&organizer); err != nil {
		return notFound(err)
	}
	if organizer != p.Subject {
		return problem(403, "EVENT_SCOPE_DENIED", "Organizer is not assigned to this event")
	}
	return nil
}
func (s *Store) fact(ctx context.Context, tx pgx.Tx, eventType, aggregate, actor, correlation string, payload map[string]any) error {
	raw, _ := json.Marshal(payload)
	_, err := tx.Exec(ctx, `INSERT INTO outbox(event_id,event_type,schema_version,occurred_at,aggregate_id,correlation_id,causation_id,actor_id,routing_key,payload) VALUES($1,$2,1,now(),$3,$4,$5,$6,$7,$8)`, uuid.NewString(), eventType, aggregate, correlation, correlation, actor, "matchmaking."+eventType, raw)
	return err
}
func (s *Store) audit(ctx context.Context, tx pgx.Tx, actor, target, action, reason string, prior, next map[string]any, correlation string) error {
	before, _ := json.Marshal(prior)
	after, _ := json.Marshal(next)
	_, err := tx.Exec(ctx, `INSERT INTO audit_log(audit_id,actor_id,target_id,action,reason,prior_state,new_state,correlation_id,occurred_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,now())`, uuid.NewString(), actor, target, action, reason, before, after, correlation)
	return err
}
func pairKey(a, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}
func problem(status int, code, detail string) *domain.ProblemError {
	return &domain.ProblemError{Status: status, Code: code, Detail: detail}
}
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return problem(404, "MATCHING_RESOURCE_NOT_FOUND", "Matching resource was not found")
	}
	return err
}
func Debug(err error) string { return fmt.Sprint(err) }
