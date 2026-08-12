DROP INDEX IF EXISTS uniq_fee_payment_open_per_head;

ALTER TABLE registration_windows DROP CONSTRAINT IF EXISTS registration_windows_time_order;
ALTER TABLE timetable_slots DROP CONSTRAINT IF EXISTS timetable_slots_time_order;
ALTER TABLE fee_payments DROP CONSTRAINT IF EXISTS fee_payments_amount_non_negative;
ALTER TABLE fee_heads DROP CONSTRAINT IF EXISTS fee_heads_amount_non_negative;
ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_credits_non_negative;
ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_seat_cap_positive;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_slug_not_reserved;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_slug_format;

DROP FUNCTION IF EXISTS tenant_usage_since(DATE);
DROP FUNCTION IF EXISTS outbox_dead_letter_count();
DROP FUNCTION IF EXISTS outbox_mark_failed(UUID, TEXT, INT);
DROP FUNCTION IF EXISTS outbox_mark_published(UUID);
DROP FUNCTION IF EXISTS outbox_claim_pending(INT, INT);

DROP INDEX IF EXISTS idx_outbox_pending;
ALTER TABLE outbox
  DROP COLUMN IF EXISTS failed_at,
  DROP COLUMN IF EXISTS last_error,
  DROP COLUMN IF EXISTS attempts;

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
         USING (tenant_id = current_setting(''app.tenant_id'', true)::uuid)',
      t
    );
  END LOOP;
END
$$;
