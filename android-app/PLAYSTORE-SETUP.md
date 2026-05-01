# Play Store onboarding — one-time setup

This walkthrough is the **one-time onboarding** to get the app
listed on Google Play and the GitHub Actions release pipeline
authenticated against it. After this is done once, every release
is just `git tag android-vX.Y.Z && git push origin android-vX.Y.Z`
— see [`RELEASE.md`](./RELEASE.md) for the day-to-day mechanics.

Order matters: you can't get the service-account JSON until the
Play Console listing exists, and the listing won't accept API
uploads until you've done one manual upload to seed it.

## Five GitHub secrets you'll end up with

Set them under **Repo → Settings → Secrets and variables → Actions**.

| Secret | Origin | Format |
|---|---|---|
| `ANDROID_KEYSTORE_BASE64` | Block 1, step 2 | Single-line base64 |
| `ANDROID_KEYSTORE_PASSWORD` | Block 1, step 1 (you choose) | Plain text |
| `ANDROID_KEY_ALIAS` | Block 1, step 1 (`upload`) | Plain text |
| `ANDROID_KEY_PASSWORD` | Block 1, step 1 (you choose) | Plain text |
| `PLAY_STORE_SERVICE_ACCOUNT_JSON` | Block 3, step 5 | Raw JSON, no base64 |

---

## Block 1 — Generate the upload keystore (15 min, fully local)

You can do this immediately; nothing on Google's side is needed.

### Step 1. Generate the keystore

```bash
cd ~                                 # somewhere OUTSIDE the repo
keytool -genkey -v \
  -keystore upload.keystore \
  -alias upload \
  -keyalg RSA -keysize 2048 \
  -validity 9125 \
  -storepass <STORE_PW> -keypass <KEY_PW> \
  -dname "CN=ScreenTime, O=Personal, C=GB"
```

  - `<STORE_PW>` and `<KEY_PW>` — pick passwords. Using the same
    value for both is fine. **Save them in a password manager
    immediately**; there is no recovery path.
  - `-validity 9125` ≈ 25 years. Once you publish to Play Store the
    key fingerprint is permanent for `com.nigel4321.macosscreentime`.
  - `-dname` — the cert subject, never shown to users; `C=GB` is
    fine if you're in the UK.

### Step 2. Back the file up

**Critical**: copy `upload.keystore` to:

  - your password manager as a file attachment; **and**
  - one off-machine location (encrypted USB stick, cloud drive
    with 2FA, hardware-backed backup).

If you lose this file, you can never publish another update under
this `applicationId` for the next 25 years. Google does **not**
support key replacement for upgrade builds without a brand-new
listing.

### Step 3. Base64-encode for the GitHub secret

```bash
base64 -w 0 upload.keystore | xclip -selection clipboard   # Linux
base64 -w 0 upload.keystore | pbcopy                       # macOS
# or to a file:
base64 -w 0 upload.keystore > upload.keystore.b64
```

`-w 0` keeps the output on a single line — GitHub Secrets accepts
either form but a single line copies cleanly.

### Step 4. Add four GitHub secrets

| Secret | Value |
|---|---|
| `ANDROID_KEYSTORE_BASE64` | The base64 output from step 3 |
| `ANDROID_KEYSTORE_PASSWORD` | `<STORE_PW>` from step 1 |
| `ANDROID_KEY_ALIAS` | `upload` |
| `ANDROID_KEY_PASSWORD` | `<KEY_PW>` from step 1 |

---

## Block 2 — Play Console listing + first manual upload (1 hr+, requires Google)

This is the part Google forces to be manual.

### Step 1. Sign up for the Play Console

  - <https://play.google.com/console> — $25 one-time fee.
  - Real-name registration with photo ID.
  - Verification usually clears in < 24h, occasionally a few days.

### Step 2. Create the app

**Play Console → Create app**:

  - App name: `ScreenTime` (or whatever you want shown in the
    store)
  - Default language: English (UK or US)
  - App or game: **App**
  - Free or paid: **Free**
  - Tick the developer-program-policies + US-export-laws
    declarations
  - Click **Create app**

### Step 3. Fill the minimum listing

Walk **Dashboard → Set up your app**. The fields required to
upload anything (even to internal testing) are:

  - **App access** — the app needs an account. Either provide demo
    credentials or describe the gating; reviewers won't be able to
    log in otherwise.
  - **Ads** — declare **No**.
  - **Content rating** — fill the questionnaire. Self-hosted
    parental controls with no chat / UGC / external links → all
    "No" works; rating comes back as "Everyone" / PEGI 3.
  - **Target audience** — pick adult age ranges; this is **not** a
    children's app.
  - **Data safety** — declare what is collected. Match the privacy
    policy below or Play Console rejects mismatches:
      - Email address — collected, linked to user, required
      - App activity — collected, linked to user, required
      - Device or other identifiers — collected, linked to user,
        required
      - **Encrypted in transit**: yes; **deletable on request**:
        yes.
  - **Government apps** / **Financial features** / **Health** /
    **News** / **COVID-19 contact tracing** — all **No**.
  - **Privacy policy URL** — required. Use the policy at
    `docs/privacy.md` in this repo. Once GitHub Pages is enabled
    on the `main` branch with **Source: /docs**, it'll be served
    at:

    <https://nigel4321.github.io/macos-screentime/privacy>

    (Repo → Settings → Pages → "Deploy from a branch" → branch
    `main`, folder `/docs`. First build takes ~1 minute.)

### Step 4. Main store listing assets

  - **Short description** (≤ 80 chars) — e.g. _"Time-track and
    limit app usage on family Macs from your Android phone."_
  - **Full description** (≤ 4000 chars).
  - **App icon** — 512×512 PNG, ≤ 1 MB.
  - **Feature graphic** — 1024×500 PNG, ≤ 1 MB.
  - **Phone screenshots** — at least 2, between 320–3840 px on the
    long side, 16:9 or 9:16. Roborazzi screenshots from CI work
    fine; download `today-screenshots-<sha>` from a recent
    `android` workflow run as ZIP.

### Step 5. Build a signed AAB locally

```bash
cd ~/macos-screentime/android-app
GITHUB_REF_NAME=android-v0.1.0 \
RELEASE_KEYSTORE_PATH=$HOME/upload.keystore \
RELEASE_KEYSTORE_PASSWORD=<STORE_PW> \
RELEASE_KEY_ALIAS=upload \
RELEASE_KEY_PASSWORD=<KEY_PW> \
./gradlew :app:bundleRelease
```

Output: `app/build/outputs/bundle/release/app-release.aab`
(~4–5 MB, `versionCode=1000`, `versionName=0.1.0`).

### Step 6. Manually upload it (this seeds the listing)

  - **Play Console → your app → Testing → Internal testing →
    Create new release → Upload** the AAB.
  - Add a release note (any text — e.g. _"First internal seed
    build."_)
  - Click **Save** (NOT _Review release_ — we're just seeding the
    listing).

The listing now exists with a versionCode the API will accept
above. Google **will not** accept API uploads to a brand-new
listing.

---

## Block 3 — Service account for API uploads (30 min)

### Step 1. Link a GCP project to Play Console

**Play Console → Settings (gear, top-left) → API access**.

  - Click **Choose to enable the API and link the project**.
  - Pick **Create new project** rather than reusing an existing
    GCP project — Play Console expects exclusive ownership.
  - Wait for the link to complete (~30s).

### Step 2. Create the service account in GCP

  - From Play Console → API access → click **Learn how to create
    service accounts**, which deep-links to GCP.
  - Or go directly to <https://console.cloud.google.com/> → pick
    the project Play Console just linked → **IAM & Admin →
    Service accounts → Create service account**:
      - Name: `play-publisher`
      - Skip "Grant this service account access to project" (no
        GCP role needed — only the Play Console role matters).
      - **Done**.

### Step 3. Generate a JSON key

On the new service-account row → **⋮ menu → Manage keys → Add key
→ Create new key → JSON**. A JSON file downloads. **This is the
secret.** Treat it like a password.

### Step 4. Grant Play Console permissions

Back in **Play Console → Settings → API access**. The new service
account now appears under "Service accounts". Click **Manage Play
Console permissions** next to it.

  - **App permissions** tab → **Add app**: pick
    `com.nigel4321.macosscreentime`. Don't grant org-wide.
  - **Permissions**:
      - **Releases → Manage testing tracks** ✓ (covers
        internal/alpha/beta)
      - **Releases → Release apps to production** ✓ (you'll need
        this when you ship to production; no harm having it now)
      - **App information → View app information and download
        bulk reports** ✓ (`supply` reads listing metadata)
  - Click **Invite user → Send invitation** (auto-accepted for
    service accounts).

> Permissions take **5–60 minutes to propagate**. The first
> workflow run after this step can fail with `The caller does not
> have permission`; wait 10 min and re-run.

### Step 5. Add the fifth GitHub secret

  - Open the JSON file you downloaded.
  - Copy its **entire raw contents**, including the curly braces.
    No base64.
  - GitHub: **Settings → Secrets and variables → Actions → New
    repository secret**.
  - Name: `PLAY_STORE_SERVICE_ACCOUNT_JSON`.
  - Value: paste the raw JSON.
  - Save.

---

## Block 4 — Fire it for real

Once all five secrets are in place:

```bash
cd ~/macos-screentime
git tag android-v0.1.1            # bump from the manual upload's 0.1.0
git push origin android-v0.1.1
```

`.github/workflows/android-release.yml` will:

  1. decode the keystore from `ANDROID_KEYSTORE_BASE64`
  2. write the service-account JSON to disk
  3. build a signed AAB with `versionCode=1001`, `versionName=0.1.1`
  4. upload it to the **internal-testing** track via
     `fastlane internal`
  5. attach the AAB as a workflow artifact for inspection

The release lands as a **draft** in Play Console. You still have
to click **Roll out to internal testing** in the Play Console UI
before testers see it — we never auto-publish.

To **promote a tag up the alpha → beta → production ladder**:

  - **Actions tab → android-release → Run workflow**.
  - Pick the tag in the _Use workflow from_ dropdown.
  - Pick the `track` input (alpha / beta / production).

Each track also lands as a draft.

---

## Common gotchas

- **versionCode collision**: you've already used versionCode 1000
  (from the manual `android-v0.1.0` upload). Tag the next release
  `android-v0.1.1` (= 1001) or higher. Play Console rejects
  re-uploads of a versionCode it has already seen.
- **Service-account permission propagation** — see the box in
  Block 3 step 4.
- **Privacy policy URL must be reachable** before any release
  leaves draft. Internal testing tolerates a brief 404 window
  while Pages is building, but production review will reject a
  missing or 404 URL.
- **Two-factor on Play Console**: enable it. The developer account
  is the single most valuable login the project has.
