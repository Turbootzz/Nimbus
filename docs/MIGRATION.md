# Migrating from Deprecated to Unified Nimbus

> **Note**: Both `docker-compose.yml` (unified) and `docker-compose.deprecated.yml` (separate containers) are **currently supported**. The deprecated setup remains available for existing users, though we recommend migrating to the unified setup when convenient. Migration is optional!

## Should You Migrate?

### Reasons to Stay on Deprecated Setup (For Now)
- Already deployed and working well - "if it ain't broke, don't fix it"
- Using Portainer with custom networking that works
- Need separate backend/frontend containers for specific requirements
- Existing reverse proxy setup you don't want to reconfigure
- Time constraints - can migrate later when convenient

### Reasons to Migrate to Unified
- **Simpler deployment** - 2 containers instead of 3
- **Zero CORS configuration** - Internal nginx proxy handles routing
- **Easier environment setup** - Fewer variables to configure
- **Recommended for new users** - Less complexity
- **Faster updates** - Single image to pull

---

## Quick Migration Overview (5-10 minutes)

### What You'll Need
- [ ] Access to current .env file or environment variables
- [ ] 5-10 minutes of downtime acceptable
- [ ] Backup of Docker volumes (recommended)
- [ ] OAuth provider access (if using OAuth login)

### Critical Information to Preserve
1. **`JWT_SECRET`** - If changed, **all users will be logged out**
2. **`DB_PASSWORD`** - Must match database password
3. **OAuth redirect URLs** - Need to be updated in provider dashboards (port changes from 8080 to 3000)

### What Happens Automatically
- **PostgreSQL data migration** - The postgres entrypoint automatically detects and migrates legacy data paths

---

## Migration Guide for CLI Users

### Step 1: Backup Everything

**Stop your current stack** (keeps volumes intact):
```bash
docker-compose -f docker-compose.deprecated.yml down
```

**Create backups** (recommended):
```bash
# Backup postgres volume
docker run --rm \
  -v nimbus_postgres_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/postgres-backup-$(date +%Y%m%d-%H%M%S).tar.gz /data

# Backup uploads volume
docker run --rm \
  -v nimbus_uploads_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/uploads-backup-$(date +%Y%m%d-%H%M%S).tar.gz /data

# Backup .env file
cp .env .env.backup-$(date +%Y%m%d-%H%M%S)
```

### Step 2: Extract Critical Secrets

**From your current `.env` file**, note these critical values:

```bash
# CRITICAL - Must preserve exactly!
JWT_SECRET=your-jwt-secret-here       # Find this in your .env
DB_PASSWORD=your-db-password-here     # Find this in your .env

# Optional - If you use OAuth:
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GOOGLE_REDIRECT_URL=...

GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...
GITHUB_REDIRECT_URL=...

DISCORD_CLIENT_ID=...
DISCORD_CLIENT_SECRET=...
DISCORD_REDIRECT_URL=...
```

> **Important**: Write these down or copy them to a safe place. The `JWT_SECRET` is especially critical - if you lose it, all users will be logged out.

### Step 3: Download Unified Configuration

**Option A: Download from GitHub**
```bash
curl -O https://raw.githubusercontent.com/Turbootzz/Nimbus/main/docker-compose.yml
```

**Option B: Use existing file** (if you already have it):
```bash
# The file should already be in your nimbus directory
ls docker-compose.yml
```

### Step 4: Create New Environment Configuration

Create a new `.env` file with your preserved secrets:

```bash
cat > .env << 'EOF'
# Database Configuration
DB_PASSWORD=YOUR_DB_PASSWORD_HERE    # Paste from Step 2

# Authentication
JWT_SECRET=YOUR_JWT_SECRET_HERE      # Paste from Step 2

# Optional: OAuth Configuration (if you use it)
# Google OAuth
# GOOGLE_CLIENT_ID=
# GOOGLE_CLIENT_SECRET=
# GOOGLE_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/google/callback

# GitHub OAuth
# GITHUB_CLIENT_ID=
# GITHUB_CLIENT_SECRET=
# GITHUB_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/github/callback

# Discord OAuth
# DISCORD_CLIENT_ID=
# DISCORD_CLIENT_SECRET=
# DISCORD_REDIRECT_URL=http://localhost:3000/api/v1/auth/oauth/discord/callback

EOF
```

> **Notice**: OAuth redirect URLs now use port **3000** instead of 8080. You'll need to update these in your OAuth provider dashboards (see Step 6).

### Step 5: Start Unified Stack

```bash
docker-compose up -d
```

**Monitor startup**:
```bash
# Watch logs
docker-compose logs -f

# Check container status
docker-compose ps
```

Wait for containers to show as "healthy" (usually 30-60 seconds).

### Step 6: Verify Migration Success

```bash
# Check API health
curl http://localhost:3000/api/v1/health

# Expected response:
# {"status":"ok"}
```

**Manual verification**:
1. Open http://localhost:3000 in your browser
2. **Log in with your existing account** - if JWT_SECRET was preserved correctly, you should NOT need to reset your password
3. Verify all your services are visible
4. Check that service icons/avatars display correctly
5. Try creating a new service to confirm everything works

### Step 7: Update OAuth Redirect URLs (if using OAuth)

If you use OAuth login, you **must update** the redirect URLs in your OAuth provider dashboards. See the [OAuth Updates](#oauth-redirect-url-updates) section below.

### Step 8: Clean Up (Optional, after 1 week)

Once you've verified everything works for at least a week:

```bash
# Remove old backup files (keep them for 30 days recommended)
rm postgres-backup-*.tar.gz
rm uploads-backup-*.tar.gz
rm .env.backup-*

# Remove old deprecated docker-compose file (optional)
# mv docker-compose.deprecated.yml docker-compose.deprecated.yml.bak
```

---

## Migration Guide for Portainer Users

### Step 1: Note Current Configuration

1. **Open your deprecated Nimbus stack** in Portainer
2. **Screenshot or copy all environment variables**, especially:
   - `JWT_SECRET` (CRITICAL!)
   - `DB_PASSWORD` (CRITICAL!)
   - `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` (if using Google OAuth)
   - `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET` (if using GitHub OAuth)
   - `DISCORD_CLIENT_ID`, `DISCORD_CLIENT_SECRET` (if using Discord OAuth)
   - Any other custom variables you set

> **Tip**: Take a screenshot of the entire environment variables section for reference.

### Step 2: Backup Volumes (Recommended)

**Option A: Using Portainer UI**
1. Go to **Volumes** in Portainer
2. Find `nimbus_postgres_data`
3. Use Portainer's backup feature if available

**Option B: Using Docker commands** (from server SSH/terminal):
```bash
# Backup postgres volume
docker run --rm \
  -v nimbus_postgres_data:/data \
  -v /tmp:/backup \
  alpine tar czf /backup/nimbus-postgres-backup.tar.gz /data

# Backup uploads volume
docker run --rm \
  -v nimbus_uploads_data:/data \
  -v /tmp:/backup \
  alpine tar czf /backup/nimbus-uploads-backup.tar.gz /data
```

### Step 3: Stop Deprecated Stack (Don't Delete!)

1. In Portainer, select your deprecated Nimbus stack
2. Click **"Stop"** (⏸️ button)
3. **Do NOT click "Remove"** - this preserves your volumes and data

The stack should now show as "Stopped" but still be visible in your stack list.

### Step 4: Create New Unified Stack

1. **In Portainer, click "Add Stack"**
2. **Name it** something like `nimbus-unified` or `nimbus`
3. **Select "Web editor"** as the build method
4. **Paste the unified docker-compose.yml**:
   - Copy from: https://raw.githubusercontent.com/Turbootzz/Nimbus/main/docker-compose.yml
   - Or copy from the `docker-compose.yml` file in your project

5. **Scroll down to "Environment variables"**
6. **Add your preserved environment variables**:

   Click **"Add an environment variable"** for each:

   | Name | Value | Notes |
   |------|-------|-------|
   | `DB_PASSWORD` | (your password from Step 1) | CRITICAL |
   | `JWT_SECRET` | (your secret from Step 1) | CRITICAL |
   | `GOOGLE_CLIENT_ID` | (if using Google OAuth) | Optional |
   | `GOOGLE_CLIENT_SECRET` | (if using Google OAuth) | Optional |
   | `GOOGLE_REDIRECT_URL` | `http://localhost:3000/api/v1/auth/oauth/google/callback` | Note port 3000 |
   | `GITHUB_CLIENT_ID` | (if using GitHub OAuth) | Optional |
   | `GITHUB_CLIENT_SECRET` | (if using GitHub OAuth) | Optional |
   | `GITHUB_REDIRECT_URL` | `http://localhost:3000/api/v1/auth/oauth/github/callback` | Note port 3000 |
   | `DISCORD_CLIENT_ID` | (if using Discord OAuth) | Optional |
   | `DISCORD_CLIENT_SECRET` | (if using Discord OAuth) | Optional |
   | `DISCORD_REDIRECT_URL` | `http://localhost:3000/api/v1/auth/oauth/discord/callback` | Note port 3000 |

> **Important**: OAuth redirect URLs now use port **3000** instead of 8080!

7. **Click "Deploy the stack"**

### Step 5: Monitor Deployment

1. Wait for containers to deploy (usually 1-2 minutes)
2. Check container status - both should show as **"healthy"** (green)
3. If containers show as "unhealthy" or "starting", wait another minute
4. Check logs if issues persist:
   - Click on the stack → Click on container name → View logs

### Step 6: Verify Migration Success

1. **Open Nimbus** in your browser:
   - http://localhost:3000 (or your domain)
2. **Try logging in** with your existing account
   - If JWT_SECRET was preserved correctly, your session should still be valid OR you can log in normally
3. **Check your dashboard**:
   - ✓ All services visible
   - ✓ Service icons display correctly
   - ✓ User avatar displays (if you had one)
4. **Test functionality**:
   - Create a new service
   - Edit a service
   - Check that health monitoring still works

### Step 7: Update OAuth Redirect URLs (if using OAuth)

If you use OAuth login (Google, GitHub, or Discord), you **must update** the redirect URLs in your OAuth provider dashboards. The port changed from **8080 → 3000**.

See the [OAuth Updates](#oauth-redirect-url-updates) section below for detailed instructions.

### Step 8: Clean Up (After 1 Week)

Once you've verified everything works for at least a week:

1. **Delete the old deprecated stack** in Portainer:
   - Select the old stack → Click "Remove"
   - Volumes will NOT be deleted (they're now used by the new stack)
2. **Keep backups for 30 days** (just in case)

---

## What Changes vs What Stays the Same

| Aspect | Deprecated (Old) | Unified (New) | Impact |
|--------|------------------|---------------|--------|
| **Number of containers** | 3 (db, backend, frontend) | 2 (db, nimbus) | Simpler architecture |
| **Frontend URL** | `http://localhost:3000` | `http://localhost:3000` | ✅ **No change** |
| **API endpoint** | `http://localhost:8080/api/v1` | `http://localhost:3000/api/v1` | ⚠️ Port changed |
| **Database** | PostgreSQL 18 | PostgreSQL 18 | ✅ Auto-migrates data |
| **All user data** | Preserved | Preserved | ✅ **No data loss** |
| **JWT_SECRET** | In .env | Must copy to new .env | ⚠️ **CRITICAL** |
| **DB_PASSWORD** | In .env | Must copy to new .env | ⚠️ **CRITICAL** |
| **CORS configuration** | Required (`CORS_ORIGINS`) | Not needed | ✅ Simplified |
| **API URL config** | Required (`NEXT_PUBLIC_API_URL`) | Not needed | ✅ Simplified |
| **Upload volume path** | `/app/uploads` | `/app/backend/uploads` | Usually auto-handles |
| **OAuth redirect port** | `:8080` | `:3000` | ⚠️ Must update providers |

### Key Takeaways

✅ **Automatic**: PostgreSQL data migration
✅ **Preserved**: All users, services, settings, uploads
✅ **Simplified**: Less environment configuration needed
⚠️ **Action Required**: Copy JWT_SECRET and DB_PASSWORD
⚠️ **Action Required**: Update OAuth redirect URLs (if using OAuth)

---

## OAuth Redirect URL Updates

If you use OAuth login (Google, GitHub, or Discord), you **must update** the redirect URLs in your OAuth provider dashboards because the API port changed from **8080 → 3000**.

### Pattern for All Providers

**Old**: `http://localhost:8080/api/v1/auth/oauth/{provider}/callback`
**New**: `http://localhost:3000/api/v1/auth/oauth/{provider}/callback`

**For production domains**, replace `localhost:3000` with your actual domain:
- Example: `https://nimbus.yourdomain.com/api/v1/auth/oauth/{provider}/callback`

---

### Google OAuth

1. **Go to** [Google Cloud Console](https://console.cloud.google.com/)
2. **Navigate to**: APIs & Services → Credentials
3. **Select** your OAuth 2.0 Client ID (the one used for Nimbus)
4. **Find "Authorized redirect URIs"** section
5. **Remove old URI**:
   ```
   http://localhost:8080/api/v1/auth/oauth/google/callback
   ```
6. **Add new URI**:
   ```
   http://localhost:3000/api/v1/auth/oauth/google/callback
   ```
7. **For production**, also add your domain:
   ```
   https://nimbus.yourdomain.com/api/v1/auth/oauth/google/callback
   ```
8. **Click "Save"**

---

### GitHub OAuth

1. **Go to** [GitHub Developer Settings](https://github.com/settings/developers)
2. **Click** on "OAuth Apps"
3. **Select** your Nimbus OAuth App
4. **Update "Authorization callback URL"**:

   **Change from**:
   ```
   http://localhost:8080/api/v1/auth/oauth/github/callback
   ```

   **Change to**:
   ```
   http://localhost:3000/api/v1/auth/oauth/github/callback
   ```

5. **For production**, use your domain:
   ```
   https://nimbus.yourdomain.com/api/v1/auth/oauth/github/callback
   ```
6. **Click "Update application"**

---

### Discord OAuth

1. **Go to** [Discord Developer Portal](https://discord.com/developers/applications)
2. **Select** your Nimbus application
3. **Go to** OAuth2 → General
4. **Find "Redirects"** section
5. **Remove old redirect**:
   ```
   http://localhost:8080/api/v1/auth/oauth/discord/callback
   ```
6. **Add new redirect**:
   ```
   http://localhost:3000/api/v1/auth/oauth/discord/callback
   ```
7. **For production**, also add your domain:
   ```
   https://nimbus.yourdomain.com/api/v1/auth/oauth/discord/callback
   ```
8. **Click "Save Changes"**

---

### Verification

After updating OAuth providers:

1. **Log out** of Nimbus
2. **Try "Login with Google/GitHub/Discord"** button
3. **Should redirect** to provider and back successfully
4. **If you get "redirect_uri_mismatch"** error:
   - Double-check the redirect URL in your OAuth provider dashboard
   - Make sure you saved the changes
   - Verify the port is **3000** (not 8080)

---

## Troubleshooting

### All Users Are Logged Out After Migration

**Cause**: `JWT_SECRET` was not preserved or was changed

**Solution**:
1. Check your new `.env` file has the **exact same** `JWT_SECRET` as before
2. Compare with backup: `cat .env.backup-* | grep JWT_SECRET`
3. If secrets don't match, update `.env` with the correct JWT_SECRET
4. Restart: `docker-compose restart`

**If you lost the old JWT_SECRET**:
- Users will need to log in again
- Their accounts and data are safe
- Consider notifying users about the required re-login

---

### OAuth Login Fails with "redirect_uri_mismatch"

**Cause**: OAuth redirect URLs not updated in provider dashboard

**Solution**:
1. Follow the [OAuth Redirect URL Updates](#oauth-redirect-url-updates) section above
2. Make sure the redirect URL uses port **3000** (not 8080)
3. Verify you clicked "Save" in the provider dashboard
4. Wait 1-2 minutes for changes to propagate
5. Try logging in again

---

### Database Connection Failed

**Cause**: `DB_PASSWORD` doesn't match or is missing

**Solution**:
```bash
# Check current .env has DB_PASSWORD
grep DB_PASSWORD .env

# Compare with backup
grep DB_PASSWORD .env.backup-*

# If they don't match, update .env with correct password
# Then restart:
docker-compose restart
```

**Check database logs**:
```bash
docker-compose logs db
```

---

### Database Password Special Characters

**As of v1.1.1**, Nimbus fully supports special characters in database passwords. Passwords are automatically URL-encoded when constructing connection strings.

**If you're upgrading from an older version**:
- You can now use secure passwords with special characters like `!@#$%^&*()`
- No need to avoid special characters anymore
- The old warning in `.env.example` has been removed

**Using `DB_URL` directly?**

If you're setting the `DB_URL` environment variable directly (instead of using individual `DB_PASSWORD`, `DB_HOST`, etc.), you must manually URL-encode special characters in the password.

**Recommended approach**: Use individual env vars for automatic encoding:
```bash
DB_HOST=db
DB_PORT=5432
DB_USER=nimbus
DB_PASSWORD=your-password-with-special-chars!@#
DB_NAME=nimbus
```

**Alternative**: If you must use `DB_URL`, encode the password manually:
```bash
# Password: my@pass!word
# Encoded:  my%40pass%21word
DB_URL=postgres://nimbus:my%40pass%21word@db:5432/nimbus?sslmode=disable
```

**URL encoding reference**:
- `@` → `%40`
- `!` → `%21`
- `#` → `%23`
- `$` → `%24`
- `%` → `%25`
- `^` → `%5E`
- `&` → `%26`
- `*` → `%2A`
- Space → `+` or `%20`

---

### Service Icons or Avatars Missing

**Cause**: Upload volume not properly migrated or path issue

**Solution 1 - Verify volumes**:
```bash
# Check that volumes exist
docker volume ls | grep nimbus

# Should see:
# nimbus_postgres_data
# nimbus_uploads_data (or similar)
```

**Solution 2 - Manually migrate uploads** (if needed):
```bash
# Create proper directory structure and copy uploads
docker run --rm \
  -v nimbus_uploads_data:/data \
  alpine sh -c "mkdir -p /data/backend/uploads && \
                if [ -d /data/service-icons ]; then \
                  cp -r /data/service-icons /data/backend/uploads/; \
                fi && \
                if [ -d /data/avatars ]; then \
                  cp -r /data/avatars /data/backend/uploads/; \
                fi"

# Restart nimbus container
docker-compose restart nimbus
```

**Solution 3 - Restore from backup**:
```bash
# Stop unified stack
docker-compose down

# Restore uploads from backup
docker run --rm \
  -v nimbus_uploads_data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/uploads-backup-*.tar.gz -C /

# Start unified stack
docker-compose up -d
```

---

### Containers Unhealthy or Won't Start

**Cause**: Various (check logs for details)

**Diagnosis**:
```bash
# Check container status
docker-compose ps

# Check logs for errors
docker-compose logs

# Check specific container logs
docker-compose logs nimbus
docker-compose logs db
```

**Common solutions**:

1. **Missing environment variables**:
   ```bash
   # Verify .env has required variables
   grep -E "DB_PASSWORD|JWT_SECRET" .env
   ```

2. **Conflicting containers**:
   ```bash
   # Check for other Nimbus containers running
   docker ps | grep nimbus

   # Stop them if found
   docker stop <container-id>
   ```

3. **Database not ready**:
   ```bash
   # Database might take 30-60 seconds to be ready
   # Wait and check again
   docker-compose logs db | grep "database system is ready"
   ```

4. **Insufficient resources**:
   ```bash
   # Check Docker resources
   docker stats

   # If memory/CPU is maxed, increase Docker resource limits
   ```

---

### Containers Start but Can't Access Nimbus

**Cause**: Port conflict or firewall issue

**Solution**:
```bash
# Check if port 3000 is in use
lsof -i :3000
# or
netstat -an | grep 3000

# If port is in use by another service, either:
# 1. Stop that service, or
# 2. Change Nimbus port in docker-compose.yml:
#    ports:
#      - "3001:3000"  # Use 3001 instead
```

**Test locally**:
```bash
curl http://localhost:3000/api/v1/health

# Should return: {"status":"ok"}
```

---

## Rollback Procedure

If migration doesn't work or you need to go back to the deprecated setup:

### For CLI Users

```bash
# Step 1: Stop unified stack
docker-compose down

# Step 2: Restore .env from backup
cp .env.backup-YYYYMMDD-HHMMSS .env

# Step 3: Start deprecated stack
docker-compose -f docker-compose.deprecated.yml up -d

# Step 4: Verify it's working
docker-compose -f docker-compose.deprecated.yml ps
curl http://localhost:8080/api/v1/health
```

**If you need to restore volumes from backup**:
```bash
# Stop all containers first
docker-compose down

# Restore postgres volume
docker run --rm \
  -v nimbus_postgres_data:/data \
  -v $(pwd):/backup \
  alpine sh -c "rm -rf /data/* && tar xzf /backup/postgres-backup-*.tar.gz -C /"

# Restore uploads volume
docker run --rm \
  -v nimbus_uploads_data:/data \
  -v $(pwd):/backup \
  alpine sh -c "rm -rf /data/* && tar xzf /backup/uploads-backup-*.tar.gz -C /"

# Start deprecated stack
docker-compose -f docker-compose.deprecated.yml up -d
```

### For Portainer Users

1. **Stop the unified stack** in Portainer UI
2. **Start the old deprecated stack** (should still be in your stack list as "Stopped")
3. **Verify everything works**:
   - Access http://localhost:3000
   - Log in and check services
4. **Update OAuth redirect URLs back to `:8080`** (if you had changed them)
5. **If needed, restore from backup volumes** using the Docker commands above

---

## FAQ

### Will my users be logged out?

**No**, if you preserve `JWT_SECRET` exactly as it was. If `JWT_SECRET` changes, yes, all users will need to log in again (but their accounts and data are safe).

### Will I lose any data?

**No**. All database data (users, services, settings), uploads (service icons, avatars), and configurations are preserved during migration.

### How long is the downtime?

Typically **2-5 minutes**. Allow up to **10 minutes** to be safe, including time for verification.

### Can I test the migration first?

**Yes!** Highly recommended. Test on a development or staging environment with a copy of your production data first.

### Do I have to migrate?

**Not immediately.** The deprecated setup (`docker-compose.deprecated.yml`) is still maintained and works fine. However, we recommend migrating when convenient as the unified setup is simpler and will receive priority for new features.

### What if I use a reverse proxy (nginx, Traefik, Caddy)?

Update your reverse proxy configuration to point to **port 3000** instead of port 8080 for the backend. The unified image serves both frontend and API on port 3000.

**Example nginx config change**:
```nginx
# Before (deprecated)
location /api/v1 {
    proxy_pass http://localhost:8080;
}

# After (unified)
location / {
    proxy_pass http://localhost:3000;  # Handles both frontend and API
}
```

### Can I rollback a week (or month) later?

**Yes**, as long as you kept the backup files and the old `.env` file. See the [Rollback Procedure](#rollback-procedure) section.

### Does this work with Portainer?

**Yes!** Portainer users should follow the [Migration Guide for Portainer Users](#migration-guide-for-portainer-users) section, which has Portainer-specific steps.

### What happens to my Docker volumes?

They're **automatically reused** by the unified stack. Both setups use the same volume names (`nimbus_postgres_data`, `nimbus_uploads_data`), so your data is preserved.

### Do I need to update DNS or domain settings?

**No**, if you're using the same port 3000 for the frontend. If you had a reverse proxy pointing to backend:8080, you'll need to update it to point to the unified nimbus:3000.

### Can I run both stacks simultaneously (for testing)?

**Not recommended** with the same volumes. If you want to test side-by-side:
1. Create a new test stack with different volume names
2. Import test data
3. Test the unified setup
4. Once satisfied, migrate production

### What about my custom environment variables?

Most custom variables from the deprecated setup are no longer needed (like `CORS_ORIGINS`, `NEXT_PUBLIC_API_URL`). Only preserve:
- `JWT_SECRET` (critical)
- `DB_PASSWORD` (critical)
- OAuth credentials (if using OAuth)
- Any optional settings you explicitly configured

---

## Need Help?

If you encounter issues not covered in this guide:

1. **Check logs**: `docker-compose logs` often reveals the issue
2. **GitHub Issues**: [Report a problem](https://github.com/Turbootzz/Nimbus/issues)
3. **Discussions**: [Ask the community](https://github.com/Turbootzz/Nimbus/discussions)

When asking for help, include:
- Your setup (CLI or Portainer)
- Docker compose version
- Error messages from logs
- Steps you've already tried
