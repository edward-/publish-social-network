# Platform Setup Guide

Step-by-step instructions for connecting the publisher to Facebook, Instagram, TikTok, and YouTube.

## Table of Contents
1. [General workflow](#general-workflow)
2. [Facebook](#facebook)
3. [Instagram](#instagram)
4. [TikTok](#tiktok)
5. [YouTube](#youtube)
6. [Verifying the connection](#verifying-the-connection)
7. [Troubleshooting](#troubleshooting)

---

## General workflow

For every platform, the flow is the same:

1. Create / configure a developer app on the platform's portal.
2. Set the OAuth redirect URI to `http://localhost:8989` (must match the value the CLI uses).
3. Request the required scopes/permissions.
4. Either run the matching `login-<platform>` subcommand (Facebook, TikTok, YouTube) **or** copy a token into `.env` manually (Instagram).
5. Run `go run ./cmd/publisher publish --platforms <platform> ...` to test.

Three of the four platforms have a `login-<platform>` subcommand that opens the browser, runs the OAuth dance, and writes the token back to `.env`:

```bash
./publisher login-facebook
./publisher login-tiktok
./publisher login-youtube
```

Instagram is the exception — see the [Instagram section](#instagram) for why it's manual.

Before any of this works, you need real credentials in the developer portal. The next four sections walk through that setup per platform.

---

## Facebook

### 1. Create a Facebook App

1. Go to https://developers.facebook.com and sign in.
2. Click **My Apps → Create App**.
3. Choose **Business** as the app type (required for Graph API publishing).
4. Fill in app name, contact email, and select/create a Business portfolio.

### 2. Add the Facebook Login product

1. In your app's dashboard, scroll to **Add a Product** and click **Set Up** on **Facebook Login for Business**.
2. Under **Settings → Valid OAuth Redirect URIs**, add:
   ```
   http://localhost:8989
   ```
3. Save changes.

### 3. Request the required permissions

The publisher uses these Graph API permissions. Request them under **App Review → Permissions and Features**:

- `pages_manage_posts` — required to publish on behalf of a Page.
- `pages_read_engagement` — required to read page metadata.
- `public_profile` — basic profile read, granted by default.

If your app is in **Development** mode you can only act on your own Pages / test users. For production, submit for App Review.

### 4. Connect your Facebook Page

1. Convert your personal Facebook account into a Page admin, or create a new Page.
2. In the Graph API Explorer (https://developers.facebook.com/tools/explorer/), select your app and request `pages_manage_posts`. Generate a **Page Access Token** (not a user token) for the Page you want to publish to.
3. Note the **Page ID** from your Page's About section.

### 5. Run the login flow

```bash
./publisher login-facebook
```

This opens the browser, walks Facebook's OAuth 2.0 flow on `http://localhost:8989`, and writes `FACEBOOK_ACCESS_TOKEN` into `.env`.

If you'd rather paste a token you already have, drop it into `.env` directly:

```bash
FACEBOOK_PAGE_ID=1234567890
FACEBOOK_ACCESS_TOKEN=EAAxxxxxx
FACEBOOK_CLIENT_ID=your_app_id
FACEBOOK_CLIENT_SECRET=your_app_secret
```

### 6. Test

```bash
./publisher publish \
  --caption "Hello from the publisher!" \
  --platforms facebook
```

Expected output: a row with `facebook  OK  https://facebook.com/<post_id>`.

### Common pitfalls

- **"Invalid OAuth access token"** — the token is a *user* token, not a *Page* token. Re-generate with the Page selected in Graph API Explorer.
- **"Permissions error"** — the app is in Development mode and you're not a Page admin/developer/tester.
- **Token expired** — Facebook Page tokens are long-lived but not permanent. Re-run `login-facebook`.

---

## Instagram

Instagram publishing piggybacks on the Facebook Graph API. There is no separate Instagram developer console — you authenticate via Facebook.

### 1. Prerequisites

- An **Instagram Business** or **Creator** account (Settings → Account → Account type and tools).
- The Instagram account must be **linked to a Facebook Page** you administer (Settings → Account → Linked accounts → Facebook).

### 2. Get the Instagram User ID and Access Token

1. Go to https://developers.facebook.com/tools/explorer/.
2. Select the Facebook app you set up in the Facebook section above.
3. Click **Generate Access Token** and grant:
   - `instagram_basic`
   - `instagram_content_publish`
   - `pages_read_engagement`
4. With that user token, call the Graph API:
   ```
   GET /me/accounts?fields=instagram_business_account{id,username}
   ```
   The `instagram_business_account.id` value is your **Instagram User ID**.
5. Exchange the short-lived user token for a **long-lived Page token** that can act on the Page (and its linked Instagram account):
   ```
   GET /oauth/access_token?grant_type=fb_exchange_token
       &client_id={app-id}
       &client_secret={app-secret}
       &fb_exchange_token={short-lived-token}
   ```

### 3. Write to `.env`

```bash
INSTAGRAM_USER_ID=17841401234567890
INSTAGRAM_ACCESS_TOKEN=IGAAxxxxx
```

There is no `login-instagram` subcommand — Instagram inherits everything from your Facebook app, so the manual copy is intentional.

### 4. ⚠️ Local file uploads do not work

The Instagram Graph API requires media to be reachable at a **public URL** before publishing. The current adapter returns an error when given a local path:

```
image upload requires public URL hosting - use URL-based upload or upload to temp hosting first
```

To publish to Instagram today, host the file somewhere with a public URL (S3, GCS, etc.) and extend the adapter to feed that URL into the container-creation call. See the `uploadToTempHosting` placeholder in `internal/adapter/instagram/publisher.go`.

### 5. Test (will return an error today)

```bash
./publisher publish \
  --caption "Hello Instagram" \
  --media ./photo.jpg \
  --type image \
  --platforms instagram
```

Expected: `instagram  FAILED  ... requires public URL hosting ...`. This is the documented gap, not a misconfiguration.

---

## TikTok

### 1. Create a TikTok Developer App

1. Go to https://developers.tiktok.com and sign in.
2. Click **Manage apps → Create app**.
3. Fill in app name, description, and category.
4. Under **Products**, add **Content Posting API** and complete the review questionnaire (TikTok requires you to describe your use case).

### 2. Configure the redirect URI

1. In your app, open **Content Posting API → Login Kit → Redirect URI**.
2. Add:
   ```
   http://localhost:8989
   ```
3. Copy the **Client Key** and **Client Secret** from the app dashboard.

### 3. Request the required scopes

In the Content Posting API product page, enable:

- `user.info.basic` — read basic profile info.
- `video.upload` — upload video files.
- `video.publish` — actually publish to TikTok.

TikTok typically requires manual approval for `video.publish`. Submit the request and wait for confirmation before continuing.

### 4. Run the login flow

```bash
./publisher login-tiktok
```

The CLI:
1. Generates a PKCE verifier and challenge.
2. Starts a local callback server on `http://localhost:8989`.
3. Opens the browser to the TikTok authorization URL.
4. Waits up to 5 minutes for you to consent and be redirected back.
5. Exchanges the auth code for an access token + refresh token.
6. Writes both into `.env` (as `TIKTOK_ACCESS_TOKEN` and `TIKTOK_REFRESH_TOKEN`).

If you already have tokens, you can paste them in:

```bash
TIKTOK_CLIENT_KEY=your_client_key
TIKTOK_CLIENT_SECRET=your_client_secret
TIKTOK_ACCESS_TOKEN=act.xxxxxx
TIKTOK_REFRESH_TOKEN=rt.xxxxxx
```

### 5. Test

```bash
./publisher publish \
  --title "TikTok test" \
  --caption "Hello TikTok" \
  --media ./video.mp4 \
  --type video \
  --platforms tiktok
```

Expected: `tiktok  OK  https://www.tiktok.com/@user/video/<id>`.

### Common pitfalls

- **"Invalid scope"** — you didn't get approved for `video.publish` yet. Check the TikTok developer dashboard.
- **"Token expired"** — TikTok access tokens are short-lived. Re-run `login-tiktok` to use the refresh token, or call `Publisher.RefreshToken()` programmatically.
- **Login flow times out** — the CLI waits 5 minutes for the callback. If you walked away, restart the command.

---

## YouTube

### 1. Create a Google Cloud project

1. Go to https://console.cloud.google.com.
2. Create a new project (or select an existing one).
3. **Enable the YouTube Data API v3**:
   - APIs & Services → Library → search "YouTube Data API v3" → Enable.

### 2. Configure the OAuth consent screen

1. APIs & Services → OAuth consent screen.
2. Choose **External** (unless you have a Google Workspace org).
3. Fill in the app name, support email, and developer contact.
4. Add scopes:
   - `https://www.googleapis.com/auth/youtube.upload`
   - `https://www.googleapis.com/auth/youtube` (optional, for broader channel access)
5. Add your Google account as a **test user** while the app is in Testing mode (production requires Google's verification, which takes days).

### 3. Create OAuth 2.0 credentials

1. APIs & Services → Credentials → **Create Credentials → OAuth client ID**.
2. Application type: **Desktop app** (or **Web application** — both work; Desktop is simpler).
3. Authorized redirect URI:
   ```
   http://localhost:8989
   ```
4. Copy the **Client ID** and **Client Secret** from the resulting credential.

### 4. Run the login flow

```bash
./publisher login-youtube
```

The CLI:
1. Starts a local callback server on `http://localhost:8989`.
2. Opens the browser to Google's OAuth consent screen.
3. Waits up to 5 minutes for you to grant access and be redirected back.
4. Exchanges the code for an access token + refresh token.
5. Writes the **refresh token** into `.env` (the access token is regenerated automatically on each publish — that's the whole point of refresh tokens).

### 5. Test

```bash
./publisher publish \
  --title "YouTube test" \
  --caption "My first upload" \
  --media ./video.mp4 \
  --type video \
  --platforms youtube
```

Expected: `youtube  OK  https://youtu.be/<id>`.

The video defaults to **private**. To change that, edit `privacyStatus` in `internal/adapter/youtube/publisher.go`.

### Common pitfalls

- **"Access blocked: This app's request is invalid"** — the redirect URI doesn't exactly match. The CLI uses `http://localhost:8989` (no path, no trailing slash, no `https`).
- **"The OAuth client was not found"** — wrong Client ID/Secret, or the project lost the credential.
- **"Insufficient permissions"** — you didn't add `youtube.upload` to the consent screen scopes, or you didn't add your account as a test user.
- **Refresh token stops working** — Google sometimes issues single-use refresh tokens. Re-run `login-youtube` to get a new one.

---

## Verifying the connection

Once you've completed setup for one or more platforms:

```bash
# Make sure the binary is built
go build -o publisher ./cmd/publisher

# Test each platform individually first, then try a multi-platform publish
./publisher publish --caption "Smoke test" --platforms facebook
./publisher publish --title "Smoke" --caption "Smoke test" --media ./sample.mp4 --type video --platforms youtube,tiktok
```

A successful run prints a table with `OK` and a URL for each platform. Failures show `FAILED` followed by the error.

To see verbose HTTP traffic (useful when debugging API errors), run with `GODEBUG=http2debug=2` or set `OAUTH2_DEBUG=1` (depends on the platform's library — not all honor it).

---

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| `missing required configuration: FACEBOOK_ACCESS_TOKEN` | `.env` not loaded. Try `--env ./.env` to make sure it's pointing at the right file. |
| `401 Unauthorized` | Token expired, revoked, or for a different account. Re-run the `login-*` flow. |
| `403 Forbidden: insufficient permissions` | Missing scope, or app in Development mode acting outside its own Pages/test users. |
| `404 Not Found` | Wrong Page ID / User ID / channel ID in `.env`. Double-check. |
| Callback server fails to start | Port `8989` is in use. Stop the conflicting process or edit the `login-*` subcommand to use a different port. |
| TikTok: `invalid_grant` | Refresh token expired or revoked. Run `login-tiktok` again. |
| YouTube: `invalid_grant` on first publish | Refresh token wasn't saved (e.g., you closed the browser before the CLI could write it). Re-run `login-youtube`. |
| `post validation failed: caption is required` | You forgot `--caption` / `-c`. It's the only strictly required flag besides `--platforms`. |
| `media path is required for video posts` | You passed `--type video` but no `--media`. |

If you hit something not covered here, the error message is wrapped — chain the `Unwrap()` calls in your debugger to see the underlying platform response.
