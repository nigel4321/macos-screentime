# Android release

This repo's Android release pipeline is fastlane-driven. Pushing a
tag matching `android-v*` runs `.github/workflows/android-release.yml`,
which builds a signed AAB and uploads it to the Play Console. Tracks:
`internal` (default), `alpha`, `beta`, `production` — pick a non-
default by re-running the workflow via `workflow_dispatch` with the
`track` input.

> **First time?** The five GitHub secrets and the Play Console
> listing have to exist before any of this runs. The full
> onboarding walkthrough lives in [`PLAYSTORE-SETUP.md`](./PLAYSTORE-SETUP.md).
> Come back to this file once setup is complete.

## Versioning

Tags are parsed by `app/build.gradle.kts`:

| Tag                | versionName    | versionCode |
| ------------------ | -------------- | ----------- |
| `android-v0.1.0`   | `0.1.0`        | `1000`      |
| `android-v0.1.5`   | `0.1.5`        | `1005`      |
| `android-v0.2.0`   | `0.2.0`        | `2000`      |
| `android-v1.0.0`   | `1.0.0`        | `1000000`   |
| _(no tag, local)_  | `0.1.0-dev`    | `1`         |

Formula: `versionCode = MAJOR*1_000_000 + MINOR*1_000 + PATCH`. Each
version field is capped at 999, so we can ship up to v2.x.x without
overflowing Android's int32 versionCode ceiling.

## Required GitHub repo secrets

Set these under **Repo → Settings → Secrets and variables → Actions**.

| Secret                            | What it holds                                                              |
| --------------------------------- | -------------------------------------------------------------------------- |
| `ANDROID_KEYSTORE_BASE64`         | The upload keystore, base64-encoded — see "Generate the upload keystore". |
| `ANDROID_KEYSTORE_PASSWORD`       | Store password set when you ran `keytool`.                                |
| `ANDROID_KEY_ALIAS`               | Key alias inside the store (we use `upload`).                              |
| `ANDROID_KEY_PASSWORD`            | Key password set when you ran `keytool` (same as store pw is fine).        |
| `PLAY_STORE_SERVICE_ACCOUNT_JSON` | Raw JSON contents of the service-account key — see "Service account".     |

## Generate the upload keystore

```bash
keytool -genkey -v \
  -keystore upload.keystore \
  -alias upload \
  -keyalg RSA -keysize 2048 \
  -validity 9125 \
  -storepass <STORE_PW> -keypass <KEY_PW> \
  -dname "CN=ScreenTime, O=Personal, C=GB"
```

`-validity 9125` ≈ 25 years; once you upload to the Play Store the
key fingerprint is permanent for this app, so a long lifetime is the
forgiving choice. **Back this file up off-machine before deleting it
from your laptop** — losing it means you can never publish another
update under this `applicationId` without rolling out a brand-new
listing.

Then base64-encode it and paste into the secret:

```bash
base64 -w 0 upload.keystore | pbcopy   # macOS
base64 -w 0 upload.keystore | xclip    # Linux
```

(`-w 0` keeps the output on a single line; GitHub Secrets accepts
either form but a single line copies cleanly.)

## Service account

1. **Play Console** → _Settings_ → _API access_ → **Choose to enable
   the API and link the project**. Pick a fresh GCP project rather
   than reusing one — Play Console expects exclusive ownership.
2. In the linked GCP project: _IAM & Admin_ → _Service Accounts_ →
   **Create service account**. Name it `play-publisher`. No project
   role needed.
3. On the new service account: **Keys** → _Add key_ → _Create new
   key_ → JSON. Download the file.
4. Back in Play Console → _API access_ → **Grant access** to the new
   service account. Permissions:
   - _Releases_: `Manage testing tracks` (alpha/beta/internal)
   - _Releases_: `Release apps to production` (when you're ready to
     promote production releases)
   - _Apps_: scope to **only** `com.nigel4321.macosscreentime`.
5. Paste the JSON file's raw contents into the
   `PLAY_STORE_SERVICE_ACCOUNT_JSON` repo secret.

## First release: manual upload required

Google Play **rejects API uploads to a brand-new listing**. You have
to upload the very first AAB through the Play Console UI to seed the
listing's `applicationId`. After that, every subsequent upload —
including the first via this workflow — uses the API.

Steps for the first release:

1. Create the Play Console listing for `com.nigel4321.macosscreentime`
   (Play Console → _Create app_ → fill required short/full
   description, contact email, privacy policy URL).
2. Build a signed AAB locally:
   ```bash
   cd android-app
   GITHUB_REF_NAME=android-v0.1.0 \
   RELEASE_KEYSTORE_PATH=$PWD/upload.keystore \
   RELEASE_KEYSTORE_PASSWORD=<store_pw> \
   RELEASE_KEY_ALIAS=upload \
   RELEASE_KEY_PASSWORD=<key_pw> \
   ./gradlew :app:bundleRelease
   ```
3. Upload `app/build/outputs/bundle/release/app-release.aab` via
   _Internal testing_ → _Create new release_ → _Upload_.
4. Save (don't roll out). The listing now exists.

From here on, push an `android-v*` tag and the workflow takes over.

## Promoting between tracks

To move an existing tag's build up the ladder:

1. Open the workflow on GitHub → _Run workflow_.
2. Pick the tag in the _Use workflow from_ dropdown.
3. Pick `alpha` / `beta` / `production` in the _track_ input.

The workflow rebuilds and uploads. Every track ships as a `draft`
release — you still have to click **Roll out** in Play Console for
testers (or production users) to actually receive the build. We
don't auto-publish.

## Local fastlane

You can run the lanes locally if you have `bundle install`'d the
`Gemfile`:

```bash
cd android-app
bundle install
bundle exec fastlane android internal   # or alpha / beta / production
```

The lanes assume an AAB already exists at
`app/build/outputs/bundle/release/app-release.aab`. Run
`./gradlew :app:bundleRelease` first.
