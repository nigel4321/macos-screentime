---
title: ScreenTime — Privacy Policy
layout: default
---

# Privacy Policy for ScreenTime

_Last updated: 2026-05-01_

This is the privacy policy for **ScreenTime**, a parental-controls
application consisting of a macOS agent and an Android companion
app, with a backend deployed in the United Kingdom.

This policy describes what data the application collects, how it is
stored, who it is shared with, and how to exercise your rights over
that data.

If you have questions, contact **nsamaroo@gmail.com**.

## Who runs this service

ScreenTime is a small, independently-operated service. The
operator (the "Service Provider") is **Nigel Samaroo**,
contactable at the address above. The service is not affiliated
with Apple, Google, or any third-party employer.

For data-protection law purposes (UK GDPR, GDPR), the Service
Provider acts as **data processor** — the parent who installs the
application on their household's devices is the **data controller**
of the resulting records and decides who within their family is
monitored.

## Who this application is for

ScreenTime is intended for use by **adults monitoring devices in
their own household** — for example, a parent observing how a
child uses their family's Mac. It is **not** marketed to children,
not listed under Google Play's _Designed for Families_ programme,
and the Android dashboard has no children-specific UI.

The person who installs the app and creates the account is
expected to have legitimate authority over the devices they enrol.

## What data is collected

The application collects only what it needs to render usage
dashboards and (in a future release) enforce time policies. It
**does not** collect:

  - the contents of any application (text typed, web pages
    visited within an app, files opened);
  - screenshots, screen recordings, or microphone/camera input;
  - keystrokes;
  - location data;
  - data from any application other than the macOS agent we ship.

It **does** collect:

| Category | What it contains | Why |
|---|---|---|
| **Account identity** | Your email address, plus the opaque "subject" identifier returned by Apple Sign-In or Google Sign-In. | To distinguish your account from others'. |
| **Device records** | A hashed device token (we never store the raw token), platform (macOS / Android), and a self-reported device fingerprint string. | To bind enrolled Macs and Android devices to your account. |
| **App-usage events** | For each foreground-app session on a monitored Mac: the macOS bundle identifier (e.g. `com.google.Chrome`), the start time, and the end time. | To compute the daily and weekly totals shown in the dashboard. |
| **App display names** | The human name macOS reports for each bundle id (e.g. _"Google Chrome"_). | So the dashboard renders friendly app names instead of raw bundle ids. |
| **Server logs** | Standard HTTP request metadata (timestamp, IP, request path, response status). Retained ≤ 14 days. | To diagnose service incidents. |

The application **does not** transmit your data to any third party
for advertising, analytics, or profiling.

## Where data is stored

All persistent data is stored in **Fly.io's London (lhr) region**,
United Kingdom, on a Postgres database operated by the Service
Provider. Encryption-in-transit (TLS 1.3) is enforced for every
connection between client and backend; encryption-at-rest is
provided by Fly.io's underlying volume infrastructure.

Sign-In identity tokens are validated against Apple's and Google's
public JWKS endpoints during the sign-in flow only. We do not
share any data with Apple or Google beyond the standard OAuth
identity verification round-trip.

## How long data is kept

  - **Account identity, device records**: kept while your account
    is active.
  - **App-usage events**: kept indefinitely so historical
    dashboards continue to work, unless you request deletion.
  - **Server logs**: rotated after 14 days.

If you delete your account (see below), all account-scoped data
is removed within 30 days.

## Your rights (UK GDPR / GDPR)

You have the right to:

  - **access** the data the service holds about your account;
  - **correct** inaccurate data (e.g. a stale display name);
  - **delete** your account and all associated records;
  - **port** your data to a machine-readable JSON export;
  - **object** to processing or **withdraw** any consent you have
    given.

To exercise any of these rights, email **nsamaroo@gmail.com** from
the address registered with your account. We aim to respond within
30 days. If you are unsatisfied with the response, you may lodge a
complaint with the UK Information Commissioner's Office (ICO) at
<https://ico.org.uk/make-a-complaint/>.

## Security

The Service Provider applies reasonable technical and
organisational measures to protect the data, including:

  - all client-server traffic carried over TLS 1.3;
  - JWT-based authentication with short-lived tokens and rotating
    signing keys;
  - device tokens hashed before storage so a database compromise
    does not yield re-usable credentials;
  - no third-party analytics, advertising, or telemetry SDKs in
    either client.

Despite these measures, no service is perfectly secure. If you
believe an account has been compromised, email the address above.

## Changes to this policy

If material changes are made to this policy, the updated version
will be posted at this URL with a new _Last updated_ date and
existing users will be notified via the email address on file.

## Contact

  - Email: **nsamaroo@gmail.com**
  - Source code: <https://github.com/nigel4321/macos-screentime>
