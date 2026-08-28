CREATE TABLE event (
 event_id UUID PRIMARY KEY, organizer_id TEXT NOT NULL, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '',
 venue_name TEXT NOT NULL DEFAULT '', broad_location TEXT NOT NULL, venue_time_zone TEXT NOT NULL,
 starts_at TIMESTAMPTZ NOT NULL, ends_at TIMESTAMPTZ NOT NULL, registration_opens_at TIMESTAMPTZ NOT NULL,
 registration_closes_at TIMESTAMPTZ NOT NULL, price NUMERIC(12,2) NOT NULL CHECK(price>=0), currency CHAR(3) NOT NULL,
 configured_capacity INTEGER NOT NULL CHECK(configured_capacity BETWEEN 1 AND 10000), capacity_policy_version BIGINT NOT NULL DEFAULT 1,
 matching_ruleset_version TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN('DRAFT','PUBLISHED','REGISTRATION_OPEN','REGISTRATION_CLOSED','CANCELLED')),
 version BIGINT NOT NULL DEFAULT 1, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL,
 CHECK(ends_at>starts_at), CHECK(registration_closes_at>registration_opens_at), CHECK(registration_closes_at<starts_at)
);
CREATE INDEX event_discovery_idx ON event(status,starts_at DESC,event_id DESC);
CREATE INDEX event_organizer_idx ON event(organizer_id,updated_at DESC);
CREATE TABLE event_audit(audit_id UUID PRIMARY KEY,event_id UUID NOT NULL REFERENCES event(event_id),actor_id TEXT NOT NULL,action TEXT NOT NULL,reason TEXT NOT NULL DEFAULT '',prior_state JSONB NOT NULL,new_state JSONB NOT NULL,correlation_id TEXT NOT NULL,occurred_at TIMESTAMPTZ NOT NULL);
CREATE TABLE outbox(event_id UUID PRIMARY KEY,event_type TEXT NOT NULL,schema_version INTEGER NOT NULL,occurred_at TIMESTAMPTZ NOT NULL,aggregate_id TEXT NOT NULL,correlation_id TEXT NOT NULL,causation_id TEXT NOT NULL DEFAULT '',actor_id TEXT NOT NULL DEFAULT '',routing_key TEXT NOT NULL,payload JSONB NOT NULL,claimed_at TIMESTAMPTZ,published_at TIMESTAMPTZ,attempts INTEGER NOT NULL DEFAULT 0);
CREATE INDEX event_outbox_unpublished_idx ON outbox(occurred_at) WHERE published_at IS NULL;
