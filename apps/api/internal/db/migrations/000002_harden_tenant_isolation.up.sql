-- Hardens tenant isolation so RLS is a real boundary rather than a formality.
--
-- Three things are required for the policies added in 000001 to actually hold:
--   1. The runtime role must not be a superuser and must not have BYPASSRLS.
--   2. Policies must carry an explicit WITH CHECK so writes cannot place a row
--      into (or move a row to) another tenant.
--   3. Code paths that legitimately span tenants (the outbox relay) must go
--      through narrow SECURITY DEFINER functions instead of a bypass role.

-- 1. Runtime role -------------------------------------------------------------

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'unicore_app') THEN
    -- Password is rotated by infra/postgres/init.sql or the operator; a role
    -- with no usable password cannot be logged into over TCP.
    CREATE ROLE unicore_app LOGIN;
  END IF;
END
$$;

ALTER ROLE unicore_app NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;

GRANT USAGE ON SCHEMA public TO unicore_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO unicore_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO unicore_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO unicore_app;

-- schema_migrations is owned by the migrator; the runtime role never writes it.
REVOKE INSERT, UPDATE, DELETE ON schema_migrations FROM unicore_app;

-- `tenants` is a platform table with no tenant_id to scope by. The runtime role
-- needs to read it (host -> tenant resolution) and insert during signup, but it
-- must never be able to delete a tenant.
REVOKE ALL ON tenants FROM unicore_app;
GRANT SELECT, INSERT, UPDATE ON tenants TO unicore_app;

-- 2. Explicit WITH CHECK on every tenant-scoped policy -------------------------
--
-- Without WITH CHECK, Postgres reuses USING for writes. That happens to be the
-- behaviour we want, but relying on an implicit default for a security boundary
-- is exactly the kind of thing that silently breaks. NULLIF guards against
-- app.tenant_id being set to an empty string, which would otherwise raise a
-- cast error instead of returning zero rows.

DO $$
DECLARE
  t text;
BEGIN
  FOREACH t IN ARRAY ARRAY[
    'tenant_config', 'users', 'login_events', 'departments', 'students',
    'faculty', 'courses', 'registration_windows', 'enrollments', 'results',
    'attendance', 'fee_heads', 'fee_payments', 'announcements', 'documents',
    'audit_log', 'timetable_slots', 'push_subscriptions', 'outbox',
    'bulk_jobs', 'tenant_usage_daily'
  ]
  LOOP
    EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);
    EXECUTE format(
      'CREATE POLICY tenant_isolation ON %I
         USING (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid)
         WITH CHECK (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid)',
      t
    );
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
  END LOOP;
END
$$;

-- 3. Cross-tenant access for the outbox relay ---------------------------------
--
-- The relay drains queued events for every tenant, so it cannot run inside a
-- single tenant's RLS scope. Rather than giving the runtime role BYPASSRLS (which
-- would disable isolation for every other query it makes), expose exactly the two
-- operations it needs as SECURITY DEFINER functions.

ALTER TABLE outbox
  ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS last_error TEXT,
  ADD COLUMN IF NOT EXISTS failed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_outbox_pending
  ON outbox (created_at)
  WHERE published_at IS NULL AND failed_at IS NULL;

CREATE OR REPLACE FUNCTION outbox_claim_pending(p_limit INT, p_max_attempts INT)
RETURNS TABLE (id UUID, tenant_id UUID, topic TEXT, payload JSONB, attempts INT)
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
  UPDATE outbox o
  SET attempts = o.attempts + 1
  WHERE o.id IN (
    SELECT c.id FROM outbox c
    WHERE c.published_at IS NULL
      AND c.failed_at IS NULL
      AND c.attempts < p_max_attempts
    ORDER BY c.created_at
    FOR UPDATE SKIP LOCKED
    LIMIT p_limit
  )
  RETURNING o.id, o.tenant_id, o.topic, o.payload, o.attempts;
$$;

CREATE OR REPLACE FUNCTION outbox_mark_published(p_id UUID)
RETURNS VOID
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
  UPDATE outbox SET published_at = now(), last_error = NULL WHERE id = p_id;
$$;

-- Terminal failure state: the relay stops retrying and the row stays queryable
-- as a dead letter instead of vanishing.
CREATE OR REPLACE FUNCTION outbox_mark_failed(p_id UUID, p_error TEXT, p_max_attempts INT)
RETURNS VOID
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
  UPDATE outbox
  SET last_error = p_error,
      failed_at = CASE WHEN attempts >= p_max_attempts THEN now() ELSE NULL END
  WHERE id = p_id;
$$;

CREATE OR REPLACE FUNCTION outbox_dead_letter_count()
RETURNS BIGINT
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
  SELECT count(*) FROM outbox WHERE failed_at IS NOT NULL;
$$;

REVOKE ALL ON FUNCTION outbox_claim_pending(INT, INT) FROM PUBLIC;
REVOKE ALL ON FUNCTION outbox_mark_published(UUID) FROM PUBLIC;
REVOKE ALL ON FUNCTION outbox_mark_failed(UUID, TEXT, INT) FROM PUBLIC;
REVOKE ALL ON FUNCTION outbox_dead_letter_count() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION outbox_claim_pending(INT, INT) TO unicore_app;
GRANT EXECUTE ON FUNCTION outbox_mark_published(UUID) TO unicore_app;
GRANT EXECUTE ON FUNCTION outbox_mark_failed(UUID, TEXT, INT) TO unicore_app;
GRANT EXECUTE ON FUNCTION outbox_dead_letter_count() TO unicore_app;

-- Platform-wide usage rollups are read by superadmins only and have no tenant
-- context to run under, so they get the same narrow treatment.
CREATE OR REPLACE FUNCTION tenant_usage_since(p_since DATE)
RETURNS SETOF tenant_usage_daily
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
  SELECT * FROM tenant_usage_daily WHERE day >= p_since ORDER BY day DESC, tenant_id;
$$;

REVOKE ALL ON FUNCTION tenant_usage_since(DATE) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION tenant_usage_since(DATE) TO unicore_app;

-- 4. Constraints that stop bad data at the boundary ---------------------------

-- Subdomains become hostnames, so restrict them to a DNS-safe shape and block
-- the labels the platform reserves for itself.
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_slug_format;
ALTER TABLE tenants ADD CONSTRAINT tenants_slug_format
  CHECK (slug ~ '^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$');

ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_slug_not_reserved;
ALTER TABLE tenants ADD CONSTRAINT tenants_slug_not_reserved
  CHECK (slug NOT IN (
    'www', 'api', 'app', 'admin', 'auth', 'login', 'static', 'assets', 'cdn',
    'mail', 'smtp', 'ftp', 'ns1', 'ns2', 'status', 'docs', 'help', 'support',
    'blog', 'dashboard', 'internal', 'metrics', 'grafana', 'prometheus',
    'test', 'staging', 'dev', 'demo', 'unicore'
  ));

ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_seat_cap_positive;
ALTER TABLE courses ADD CONSTRAINT courses_seat_cap_positive CHECK (seat_cap >= 0);

ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_credits_non_negative;
ALTER TABLE courses ADD CONSTRAINT courses_credits_non_negative CHECK (credits >= 0);

ALTER TABLE fee_heads DROP CONSTRAINT IF EXISTS fee_heads_amount_non_negative;
ALTER TABLE fee_heads ADD CONSTRAINT fee_heads_amount_non_negative
  CHECK (amount >= 0 AND late_fee_amount >= 0);

ALTER TABLE fee_payments DROP CONSTRAINT IF EXISTS fee_payments_amount_non_negative;
ALTER TABLE fee_payments ADD CONSTRAINT fee_payments_amount_non_negative CHECK (amount >= 0);

ALTER TABLE timetable_slots DROP CONSTRAINT IF EXISTS timetable_slots_time_order;
ALTER TABLE timetable_slots ADD CONSTRAINT timetable_slots_time_order CHECK (end_time > start_time);

ALTER TABLE registration_windows DROP CONSTRAINT IF EXISTS registration_windows_time_order;
ALTER TABLE registration_windows ADD CONSTRAINT registration_windows_time_order
  CHECK (closes_at > opens_at);

-- A student may hold at most one non-refunded payment per fee head. This makes
-- double-charging a constraint violation rather than something the application
-- has to remember to check.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_fee_payment_open_per_head
  ON fee_payments (tenant_id, student_id, fee_head_id)
  WHERE status IN ('pending', 'paid');
