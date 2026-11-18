-- Without these columns a token is either read-only or full access and never
-- expires. Tokens that cannot be expressed that way are removed, so a
-- rollback never widens a narrow token or revives an expired one.
DELETE FROM api_tokens
WHERE expires_at IS NOT NULL
   OR scopes NOT IN (
        'read:services read:metrics read:widgets',
        'read:services read:metrics read:widgets write:services'
   );
ALTER TABLE api_tokens DROP COLUMN IF EXISTS expires_at;
ALTER TABLE api_tokens DROP COLUMN IF EXISTS scopes;
