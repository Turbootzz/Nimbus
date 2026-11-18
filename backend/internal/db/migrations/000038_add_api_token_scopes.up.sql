-- Scopes are space-separated (e.g. 'read:services mcp'); expires_at NULL means never.
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS scopes TEXT NOT NULL DEFAULT '';
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP;

-- Existing tokens keep their access: read-only tokens get the read scopes,
-- the others can also write. No existing token gets the mcp scope.
-- read_only stays in sync with the scopes so an older version keeps working.
UPDATE api_tokens
SET scopes = CASE
    WHEN read_only THEN 'read:services read:metrics read:widgets'
    ELSE 'read:services read:metrics read:widgets write:services'
END
WHERE scopes = '';
