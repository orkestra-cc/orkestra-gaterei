# OAuth Authentication Setup Guide

This guide walks through registering OAuth 2.0 / OpenID Connect apps for the four providers Orkestra supports: **Google, Apple, GitHub, and Discord**.

> **Where credentials live.** As of the auth module ConfigService refactor (PR-C), OAuth client IDs, secrets, and redirect URLs are **runtime configuration** owned by the `auth` module. The recommended workflow is:
> 1. (Optional) Seed env vars on a fresh install — the auth module reads `OAUTH_<PROVIDER>_*` on first boot and writes them into `module_configs` in MongoDB.
> 2. Manage them from then on at `/admin/modules/auth` (operator console). Secrets are AES-256-GCM encrypted at rest.
>
> Env-var names follow the convention `OAUTH_<PROVIDER>_<FIELD>` (e.g. `OAUTH_GOOGLE_CLIENT_ID`). Older docs that show `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET` without the `OAUTH_` prefix are stale.
>
> See [`backend/internal/core/auth/CLAUDE.md`](../backend/internal/core/auth/CLAUDE.md#oauth-provider-config) for the canonical field-by-field schema.

## Prerequisites

- Access to the respective provider developer portals (Google Cloud Console, Apple Developer, GitHub, Discord)
- Orkestra backend running and reachable at your operator/client hosts (see [Multi-Environment-Setup.md](Multi-Environment-Setup.md))

## Google OAuth Setup

### Step 1: Access Google Cloud Console

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Sign in with your Google account
3. Create a new project or select an existing one

### Step 2: Enable Required APIs

1. Navigate to **APIs & Services** → **Library**
2. Search for and enable:
   - Google+ API
   - Google Identity Toolkit API

### Step 3: Configure OAuth Consent Screen

1. Go to **APIs & Services** → **OAuth consent screen**
2. Choose **External** user type (unless for internal organization)
3. Fill in the required information:
   - **App name**: Orkestra
   - **User support email**: Your support email
   - **App logo**: Upload your logo (optional)
   - **Application home page**: `https://yourdomain.com`
   - **Authorized domains**: Add your domain(s)
   - **Developer contact information**: Your email
4. Add scopes:
   - `openid`
   - `email`
   - `profile`
5. Save and continue

### Step 4: Create OAuth 2.0 Credentials

1. Navigate to **APIs & Services** → **Credentials**
2. Click **Create Credentials** → **OAuth client ID**
3. Select **Web application** as the application type
4. Configure:
   - **Name**: Orkestra Web Client
   - **Authorized JavaScript origins**:
     - `https://console.orkestra.com` (operator, prod)
     - `https://api.orkestra.com` (client, prod)
     - `http://console.localhost:8080` (dev, operator)
     - `http://client.localhost:8081` (dev, client)
   - **Authorized redirect URIs** (callbacks land on the **backend**, not the frontend):
     - `https://console.orkestra.com/v1/auth/oauth/google/callback`
     - `https://api.orkestra.com/v1/auth/oauth/google/callback`
     - `http://console.localhost:3000/v1/auth/oauth/google/callback` (dev, operator)
     - `http://api.localhost:3000/v1/auth/oauth/google/callback` (dev, client)
5. Click **Create**
6. Save the **Client ID** and **Client Secret** — paste them into `/admin/modules/auth` under the **Google** group (or set `OAUTH_GOOGLE_CLIENT_ID` / `OAUTH_GOOGLE_CLIENT_SECRET` as seed env vars on first boot).

### Step 5: Mobile App Configuration (if needed)

1. Create another OAuth client ID
2. Select **iOS** or **Android** as the application type
3. For iOS:
   - Enter your Bundle ID
4. For Android:
   - Enter your Package name
   - Enter your SHA-1 certificate fingerprint
5. Save the configuration

## Apple Sign In Setup

### Step 1: Access Apple Developer Account

1. Go to [Apple Developer](https://developer.apple.com/)
2. Sign in with your Apple ID
3. Ensure you have an active Developer Program membership

### Step 2: Register App ID

1. Navigate to **Certificates, Identifiers & Profiles**
2. Click **Identifiers** → **+** button
3. Select **App IDs** and click **Continue**
4. Select **App** and click **Continue**
5. Fill in:
   - **Description**: Orkestra
   - **Bundle ID**: Choose **Explicit** and enter `com.yourdomain.orkestra`
6. Under **Capabilities**, enable:
   - **Sign In with Apple**
7. Click **Continue** and **Register**

### Step 3: Create Service ID (for Web)

1. In **Identifiers**, click **+** button
2. Select **Services IDs** and click **Continue**
3. Fill in:
   - **Description**: Orkestra Web Service
   - **Identifier**: `com.yourdomain.orkestra.web`
4. Click **Continue** and **Register**
5. Click on the created Service ID to configure
6. Enable **Sign In with Apple**
7. Click **Configure** next to Sign In with Apple
8. Set:
   - **Primary App ID**: Select your App ID from Step 2
   - **Domains and Subdomains**: Add your operator + client hosts (e.g. `console.orkestra.com`, `api.orkestra.com`). Apple does not accept `localhost`; use ngrok for local development.
   - **Return URLs**: Add (callbacks land on the backend):
     - `https://console.orkestra.com/v1/auth/oauth/apple/callback`
     - `https://api.orkestra.com/v1/auth/oauth/apple/callback`
9. Click **Next**, **Done**, and **Continue**
10. Click **Save**

### Step 4: Create Private Key

1. Navigate to **Keys** → **+** button
2. Enter a **Key Name**: Orkestra Auth Key
3. Enable **Sign In with Apple**
4. Click **Continue** and **Register**
5. **Download the private key** (`.p8` file)
   - **IMPORTANT**: Save this file securely, you can't download it again
6. Note the **Key ID** shown on the page

### Step 5: Gather Required Information

You'll need:

- **Team ID**: Found in your Apple Developer account (top right corner)
- **Service ID**: The identifier created in Step 3 (e.g., `com.yourdomain.orkestra.web`)
- **Key ID**: From Step 4
- **Private Key**: The `.p8` file downloaded in Step 4

## GitHub OAuth Setup

1. Go to [GitHub Developer Settings → OAuth Apps](https://github.com/settings/developers).
2. Click **New OAuth App** and fill in:
   - **Application name**: Orkestra
   - **Homepage URL**: `https://console.orkestra.com` (or your operator host)
   - **Authorization callback URL**: `https://console.orkestra.com/v1/auth/oauth/github/callback`
3. Register the app, then click **Generate a new client secret**.
4. Save the **Client ID** and **Client Secret** — paste them into `/admin/modules/auth` under the **GitHub** group (or use `OAUTH_GITHUB_CLIENT_ID` / `OAUTH_GITHUB_CLIENT_SECRET` as seed env vars).
5. To support both operator and client tiers, repeat with the client host (`https://api.orkestra.com/...`) — GitHub only allows one callback per app, so you typically register **two GitHub OAuth apps** (one per tier) and switch credentials by audience.

## Discord OAuth Setup

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications) and **New Application**.
2. Open **OAuth2 → General**. Add redirects:
   - `https://console.orkestra.com/v1/auth/oauth/discord/callback`
   - `https://api.orkestra.com/v1/auth/oauth/discord/callback`
   - `http://console.localhost:3000/v1/auth/oauth/discord/callback` (dev)
3. Click **Reset Secret** to generate a client secret.
4. Save the **Client ID** and **Client Secret** — paste them into `/admin/modules/auth` under the **Discord** group (or `OAUTH_DISCORD_CLIENT_ID` / `OAUTH_DISCORD_CLIENT_SECRET`).
5. Required scopes Orkestra requests: `identify email`.

## Environment Configuration

### Backend Configuration (seed env vars, optional)

These env vars are read **only on first boot** of a fresh install to seed `module_configs`. After that, manage values from `/admin/modules/auth`. Secrets are AES-256-GCM encrypted at rest by `ConfigService`.

```bash
# Google OAuth
OAUTH_GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
OAUTH_GOOGLE_CLIENT_SECRET=your-google-client-secret
OAUTH_GOOGLE_REDIRECT_URL=https://console.orkestra.com/v1/auth/oauth/google/callback

# Apple Sign In
OAUTH_APPLE_TEAM_ID=your-team-id
OAUTH_APPLE_CLIENT_ID=com.yourdomain.orkestra.web      # Apple Service ID
OAUTH_APPLE_KEY_ID=your-key-id
OAUTH_APPLE_PRIVATE_KEY=-----BEGIN PRIVATE KEY-----\n...   # inline PEM (preferred)
OAUTH_APPLE_PRIVATE_KEY_PATH=/app/keys/AuthKey_XXXXXX.p8   # OR file fallback
OAUTH_APPLE_REDIRECT_URL=https://console.orkestra.com/v1/auth/oauth/apple/callback

# GitHub
OAUTH_GITHUB_CLIENT_ID=your-github-client-id
OAUTH_GITHUB_CLIENT_SECRET=your-github-client-secret
OAUTH_GITHUB_REDIRECT_URL=https://console.orkestra.com/v1/auth/oauth/github/callback

# Discord
OAUTH_DISCORD_CLIENT_ID=your-discord-client-id
OAUTH_DISCORD_CLIENT_SECRET=your-discord-client-secret
OAUTH_DISCORD_REDIRECT_URL=https://console.orkestra.com/v1/auth/oauth/discord/callback

# JWT — Orkestra uses RS256 (asymmetric), NOT a shared HMAC secret
JWT_PRIVATE_KEY_PATH=/app/keys/jwt_private.pem
JWT_PUBLIC_KEY_PATH=/app/keys/jwt_public.pem
```

> Generate the RS256 key pair once per environment:
> ```bash
> openssl genrsa -out jwt_private.pem 2048
> openssl rsa -in jwt_private.pem -pubout -out jwt_public.pem
> ```

### Frontend Configuration

The browser never talks to OAuth providers directly — it redirects to the **backend** endpoint `/v1/auth/oauth/{provider}/login`, which crafts the provider authorization URL with state, scopes, and the correct redirect URL. So the frontend does not need provider client IDs.

```bash
# frontend-admin (operator console) — .env.production
VITE_API_URL=https://console.orkestra.com

# frontend-client (Tier-2 client SPA) — .env.production
VITE_API_BASE=https://api.orkestra.com
```

### Mobile Configuration

For Flutter mobile app, update:

**iOS** - `ios/Runner/Info.plist`:

```xml
<key>CFBundleURLTypes</key>
<array>
    <dict>
        <key>CFBundleURLSchemes</key>
        <array>
            <string>com.googleusercontent.apps.YOUR_GOOGLE_IOS_CLIENT_ID</string>
        </array>
    </dict>
</array>
```

**Android** - Add Google services configuration file to `android/app/google-services.json`

## Testing OAuth Integration

### Local Development Testing

1. Ensure your backend is running and reachable as both `http://console.localhost:3000` (operator) and `http://api.localhost:3000` (client) — the host mux dispatches on `Host`. Add entries to `/etc/hosts` if needed:
   ```
   127.0.0.1 console.localhost api.localhost client.localhost
   ```
2. Start the frontends — operator on `http://console.localhost:8080`, client SPA on `http://client.localhost:8081`.
3. From the operator login page, click **Sign in with Google** (or GitHub / Discord / Apple via ngrok). The browser is redirected to `http://console.localhost:3000/v1/auth/oauth/google/login`, then to Google, then back to `/v1/auth/oauth/google/callback`, which sets the operator refresh cookie and redirects to the dashboard.
4. Repeat from the client SPA to verify the client-tier flow lands the refresh cookie on the `api.*` host with `aud=client` in the JWT.

### Production Testing

1. Deploy your application to your domain
2. Ensure HTTPS is properly configured
3. Test both authentication flows
4. Verify tokens are properly generated and stored

## Security Best Practices

1. **Never commit secrets** to version control
2. **Use environment variables** for all sensitive configuration
3. **Implement HTTPS** for production deployments
4. **Rotate keys periodically**
5. **Implement rate limiting** on authentication endpoints
6. **Log authentication attempts** for security monitoring
7. **Use secure session management** with JWT tokens
8. **Implement refresh token rotation**

## Troubleshooting

### Common Google OAuth Issues

**Error: redirect_uri_mismatch**

- Ensure the redirect URI in your request matches exactly with configured URIs
- Check for trailing slashes and protocol (http vs https)

**Error: invalid_client**

- Verify Client ID and Client Secret are correct
- Check if the OAuth consent screen is properly configured

### Common Apple Sign In Issues

**Error: invalid_grant**

- Ensure the authorization code is used only once
- Verify the redirect URI matches exactly

**Error: invalid_client**

- Check Team ID, Service ID, and Key ID are correct
- Verify the private key file is accessible and valid

**JWT Token Issues**

- Ensure the private key file has correct permissions
- Verify the key hasn't expired
- Check the algorithm matches (ES256 for Apple)

## Support

For additional help:

- Google OAuth: [Google Identity Platform Documentation](https://developers.google.com/identity)
- Apple Sign In: [Sign in with Apple Documentation](https://developer.apple.com/sign-in-with-apple/)
- Orkestra Issues: Contact your system administrator or development team

## Checklist

Before going to production, ensure:

- [ ] Google OAuth credentials created and configured (operator + client hosts)
- [ ] Apple Sign In credentials created and configured (operator + client hosts; ngrok used for local)
- [ ] GitHub OAuth app(s) created — one per tier if you need both
- [ ] Discord OAuth credentials created and configured
- [ ] Credentials stored in `/admin/modules/auth` (ConfigService), not in plain `.env` in prod
- [ ] Redirect URIs registered for production hosts (`console.orkestra.com`, `api.orkestra.com`)
- [ ] RS256 keys (`jwt_private.pem` / `jwt_public.pem`) generated and `JWT_PRIVATE_KEY_PATH` / `JWT_PUBLIC_KEY_PATH` set
- [ ] HTTPS enabled on production
- [ ] OAuth tested in development environment (both operator and client tiers)
- [ ] OAuth tested in production environment
- [ ] Security best practices implemented
- [ ] Monitoring and logging configured
- [ ] Documentation shared with team
