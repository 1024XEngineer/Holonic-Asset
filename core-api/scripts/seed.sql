-- Seed development users explicitly with psql. The application does not run
-- this file on startup and no password is stored in the repository.
--
-- Example:
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 \
--     -v password_hash="$BOOTSTRAP_PASSWORD_HASH" -f scripts/seed.sql
--
-- password_hash must be a bcrypt hash generated for the deployment-specific
-- bootstrap password. Existing usernames are left untouched.

INSERT INTO users (username, password, email)
SELECT format('user%s', user_number), :'password_hash',
       format('user%s@holonicasset.com', user_number)
FROM generate_series(1, 10) AS user_number
ON CONFLICT (username) DO NOTHING;
