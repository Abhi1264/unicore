-- The slug format check accepted a one-character slug, but the host parser in
-- internal/middleware requires at least three. A slug in that gap could be
-- stored and then never resolve, leaving the institute with a tenant nobody can
-- reach. Tighten the database to match the parser, which is the stricter side.
--
-- Existing rows are reported rather than rewritten: a slug is the institute's
-- subdomain, so renaming one silently would break every bookmark and emailed
-- link that institute has already handed out. An operator has to pick the new
-- slug and communicate it.
DO $$
DECLARE
  offenders TEXT;
BEGIN
  SELECT string_agg(format('%s (%s)', slug, id), ', ')
    INTO offenders
    FROM tenants
   WHERE slug !~ '^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$';

  IF offenders IS NOT NULL THEN
    RAISE EXCEPTION
      'cannot tighten tenants_slug_format: % tenant(s) have a slug shorter than 3 characters or otherwise malformed: %',
      (SELECT count(*) FROM tenants WHERE slug !~ '^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$'),
      offenders
      USING HINT = 'Rename each tenant to a slug of 3-63 characters matching ^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$, then re-run this migration.';
  END IF;
END
$$;

ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_slug_format;
ALTER TABLE tenants ADD CONSTRAINT tenants_slug_format
  CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$');
