ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_slug_format;
ALTER TABLE tenants ADD CONSTRAINT tenants_slug_format
  CHECK (slug ~ '^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$');
