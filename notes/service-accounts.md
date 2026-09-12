# Service Accounts — work split for the Terraform provider

Upstream design: `product-management/designs/service-accounts/service_accounts_design.md`
(the "Terraform Provider" section covers **A** only).

Three independent pieces of work. Agreed order: **A now, C next session, B after that.**

## A — The provider authenticates *as* a service account

New auth modes replacing the human password grant:

- **Secret SA** — `client_id` + `client_secret`, standard OIDC client-credentials.
- **Federated SA** — `client_assertion` only; no `client_id`, no secret. Keycloak
  resolves the SA client from the assertion's `iss` + `sub`.

Touches `internal/provider/provider.go` (schema + `Configure`) and
`axual-webclient/auth.go` / `client.go`. Deprecates ROPC (`username`/`password`)
and the `authmode` field; `auth0` is dropped.

## C — The provider stops breaking when PM ships SAs

PM changes that break us on arrival:

- Group membership is written by **typed URI**. `resource_group.go:271` hardcodes
  `%s/users/%v` for every member. A `/users/` URI pointing at an SA row is
  rejected **400**.
- `/api/users`, `/api/users/{uid}` and `/users/search/findByAttributes` stop
  returning SAs. `axual_user` and the user data source silently lose them.

Not optional and not on our schedule — it breaks the day PM releases.

## B — The provider *manages* service accounts

A new `axual_service_account` resource (and probably a data source) over
`POST/GET/PATCH/DELETE /api/service-accounts` and `POST /{uid}/rotate`.
Nothing in the upstream design asks for this; it is our own GitOps requirement.
Largest surface, least specified. Open problems: the one-time secret, rotation
as a Terraform operation, and federated mode's `federatedSubject`.

---

# A1 — decisions

Session scope is **A1 only**: the auth modes. The endpoint rework the upstream
design bundles into the same section (`realm` → `tenant_short_name`, optional
`apiurl`/`authurl` with Axual Cloud defaults) is **A2**, deliberately split out —
that is where every backwards-compatibility landmine lives, and A1 is purely
additive.

## Facts confirmed with Platform Manager (`axual-flux`, 2026-09-12)

PM is on branch `feat/axpd-11914-create-service-account`, MR !1911, **open
against master**. The contract can still move.

Hop-2 federated token request — exactly three form fields, nothing else
(`ServiceAccountFederationKeycloakIT.java:285-291`):

    grant_type=client_credentials
    client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer
    client_assertion=<the externally-issued token>

No `client_id`. No `client_secret`. **No `scope`.**

Secret-mode token request (`e2e/internal/client.py:198-202`):

    grant_type=client_credentials
    client_id=<the SA clientId>
    client_secret=<the one-time secret>

Also no `scope` and no `audience`.

Consequence for us: the existing `scopes` attribute feeds **neither** SA mode.
It is ROPC-only from here, and dies with ROPC.

Federated JWT is already proven end to end upstream —
`ServiceAccountFederationKeycloakIT.java:86-92` runs Testcontainers Keycloak
26.6.4 with `--features=client-auth-federated`, a second realm as a stand-in
customer IdP, and both hops. A negative test at `:242-255` proves the
`jwt.credential.sub` binding rejects a mismatched subject.

The realm header value is the tenant `shortName`; the gateway translates it to
`X-User-Tenant-Short-Name`. What we send today is correct.

## Testing federated on our side

Do **not** port the Testcontainers harness to Go. The Keycloak-side contract is
the hard part and is already proven upstream. Our side is only "build this exact
POST body, re-read the token source on refresh, never leak the assertion" —
which an `httptest` server covers better and in milliseconds.

## Mode inference

Credentials mostly arrive by environment variable, not HCL, so inference must
read env as well as config.

- Any SA credential present (secret **or** federated) ⇒ SA mode. Leftover ROPC
  values are **ignored with a warning** naming what was ignored.
- Both SA credentials present ⇒ **hard error**. Both are new and deliberate;
  there is no legacy excuse for setting both.
- No SA credential ⇒ ROPC, with the deprecation warning.
- Always log the resolved mode and where each credential came from.

The upstream design now carries this rule and its reasoning under **Auth modes**.
This deviates from the upstream design's blanket "reject any two modes at once",
on purpose. ROPC is the thing being removed, so it loses quietly. A stale
`AXUAL_AUTH_PASSWORD` left in a pipeline must not fail the apply of a customer
who is doing exactly the migration we are asking for.

## Schema

Add `client_id` and deprecate `clientid` (superseded later in the session by the
environment-variable rule — `AXUAL_CLIENTID` breaks it, and `clientid` next to
`client_secret` reads badly). Accept both, warn on `clientid`, error when both are
set to different values. `clientid` relaxes from `Required` to `Optional`, since
federated mode sends no client id at all.

## Federated token sources

Ship **`oidc_token`** and **`oidc_token_file`** only. **`oidc_token_command` is
dropped** — not deferred, not stubbed behind a flag.

Why, from the ecosystem rather than taste:

| Provider | Exec-to-get-a-token as a *provider block* attribute? |
|---|---|
| azurerm | **No** — `oidc_token`, `oidc_token_file_path`, `oidc_request_url`/`oidc_request_token`, `ado_pipeline_service_connection_id` |
| AWS | **No** — asked for in issue #24885, **closed as not planned**; `credential_process` lives in the shared config file only |
| Google | **No** — executable-sourced credentials live in the SDK's external-account config file, gated by `GOOGLE_EXTERNAL_ACCOUNT_ALLOW_EXECUTABLES=1` |
| kubernetes / helm | **Yes**, but it implements client-go's standard exec credential plugin — a Kubernetes-wide convention, not a provider invention |

Exec-for-credentials is common, but almost always in an SDK config file shared by
every tool that reads it — not as a provider attribute. The upstream design's
`AXUAL_ALLOW_EXEC_CREDENTIALS=1` gate is copied from Google, but Google pairs
that gate with a config file, so we would be taking the gate without the thing it
guards. Our customers are azurerm customers (the design targets Entra); none of
them has ever had an exec attribute to reach for.

Keep the design's name `oidc_token_file` (azurerm says `oidc_token_file_path`,
AWS says `web_identity_token_file` — ours reads fine to both).

### One thing the upstream design does not cover

The expired-assertion failure is certain to happen to someone: a token in
`oidc_token` is minted once at job start, the Keycloak token lives 5 minutes, and
a long apply outlives the assertion. Keycloak reports this as `invalid_client` —
the *same* error it gives for a missing feature flag, a missing flow execution,
and a mismatched subject. Four causes, one useless message.

So: when a hop-2 exchange fails **after an earlier one succeeded in the same
run**, say so explicitly — the assertion was accepted at the start and is no
longer valid, the token in `oidc_token` has most likely expired, use
`oidc_token_file` or raise the realm's `fedClientAssertionMaxExp`. A few lines
that turn the most likely federated failure into a self-service fix.

## auth0 — deleted, not deprecated

`auth0` was added in 2.5.5 "for Axual Trial Environment"; 2.5.6 removed the trial
guide. It is undocumented — `docs/index.md:29-30` only ever shows
`authmode = "keycloak"`. No known customer uses it.

Deleting it removes `getCachedTokenSource`, `getTokenWithAudience`, `contains`,
and the package-level `tokenSourceCache`/`tokenSourceMu` globals, and makes the
`audience` attribute dead. That is most of `auth.go`.

Keep `authmode` in the schema and make `authmode = "auth0"` a **clear error**,
not a silent ignore — anyone still on it gets told what happened instead of an
unexplained auth failure.

## Fail fast — eager token fetch

Today's ROPC path authenticates eagerly (`auth.go:75`, inside `NewClient`, inside
`Configure`), so a bad password fails at the start of `terraform plan`. The
obvious `clientcredentials.Config.Client(ctx)` implementation is **lazy** and
would quietly lose that: a wrong secret would surface on some arbitrary resource
read partway into a plan.

So: **fetch one token eagerly in `SignIn` for every mode**, then hand back the
reusing client. One extra token request per run.

It also gives the federated error message its discriminator for free:

- eager fetch **fails** ⇒ configuration or Keycloak prerequisite problem. Name
  the four real causes behind `invalid_client`.
- eager fetch **succeeded**, a later refresh fails ⇒ nothing about the config
  changed, so the assertion expired. Name the two fixes.

Without the eager fetch those two are indistinguishable and every federated
failure gets the same unhelpful message Keycloak handed us.

## TLS — stop mutating the global transport

`client.go:38`, first line of `NewClient`:

```go
http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
```

It mutates Go's **process-wide** default transport, and it runs **before**
`SignIn` — so the token request itself, carrying the password today and the
client secret or Entra assertion after this change, goes out with certificate
verification off.

Not a new bug, but worth acting on now: it is in the path we are rewriting, and
the credential is about to become more valuable (the design permits an SA to hold
tenant-admin and to manage other SAs, so a stolen SA secret is worth much more
than a stolen human password was).

**In scope:** give our client its own `http.Transport` and stop touching
`http.DefaultTransport`. No behaviour change to our own requests — we just stop
changing a global for every other caller in the process.

**Filed separately:** verifying certificates by default with an opt-out. That is
the change that actually closes the hole, and it genuinely breaks on-prem
installs using self-signed certificates — so it needs its own deprecation
warning and its own release note, not a quiet ride along an additive change.

## Environment variables

Rule: **`AXUAL_` + the attribute name, uppercased.** Nothing in between.

| Attribute | Env var |
|---|---|
| `client_id` | `AXUAL_CLIENT_ID` |
| `client_secret` | `AXUAL_CLIENT_SECRET` |
| `oidc_token` | `AXUAL_OIDC_TOKEN` |
| `oidc_token_file` | `AXUAL_OIDC_TOKEN_FILE` |

Matches the ecosystem (`ARM_OIDC_TOKEN`, `AWS_WEB_IDENTITY_TOKEN_FILE`) and gives
users a rule to guess instead of a list to look up. The legacy `AXUAL_AUTH_USERNAME`
/ `AXUAL_AUTH_PASSWORD` pair keeps its odd `AUTH_` infix and dies with ROPC.

Add **`client_id`** and deprecate `clientid`: accept both, warn on `clientid`,
error if both are set and differ. `clientid` + `client_secret` side by side reads
badly, and `AXUAL_CLIENTID` breaks the rule above.

### Bug in the upstream design's example

The design's headline federated config does not work:

```hcl
oidc_token = "${AXUAL_TOKEN}"   # <- not valid
```

HCL has no shell-environment interpolation. That is a reference to a Terraform
object named `AXUAL_TOKEN`, so the customer gets "Reference to undeclared
resource". The working forms are `oidc_token = var.axual_token` fed by
`TF_VAR_axual_token`, or omitting the attribute and letting the provider read
`AXUAL_OIDC_TOKEN`. **Tell the design author** — every CI customer will copy it.

This also makes the env fallback the *primary* path for federated, not a
convenience.

## Where the logic lives, and what tests it

There are effectively **no unit tests in this repo**. `axual-webclient/tests/` is
dead scaffolding — a `TestMain` whose comments say "I have no idea", and a test
that 1 + 1 = 2. **Delete it**; it makes the package look covered. Every real test
is an acceptance test needing a live instance, configured from `test_config.yaml`
with a username and password — the harness is itself a ROPC consumer.

Today the logic is split badly: `provider.go:61-93` does env fallback,
`auth.go:59-84` does the mode switch. Keeping that split puts the ambiguity rules
in `Configure`, where testing them means building a `req.Config` through the
Terraform framework.

So: **one pure function in `axual-webclient`, next to `auth.go`.** It takes a
plain struct of already-extracted config values plus an env lookup, and returns
the resolved mode, the credentials, warnings, and an error. No Terraform types.
`Configure` only extracts config, calls it, and turns the warnings into
diagnostics.

Tests are then ordinary table tests with no live instance and no containers:

- mode resolution across every combination of config, env, and leftovers
- the exact form body per mode, against an `httptest` token endpoint
- refresh re-reads `oidc_token_file` when the file changes
- the eager-fetch-failed vs refresh-failed error messages

Acceptance tests stay on ROPC for now — they will emit the deprecation warning,
which does not fail them. Moving the harness onto a service account needs PM
merged and the test instance upgraded; that is a follow-up and it belongs with
**C**.

## Documentation

PM has explicitly delegated the migration guidance to us
(`axual-flux openspec/changes/add-service-accounts/tasks.md:189-191`): "NOT
PLATFORM MANAGER — consumer migration guidance ... belongs with that work, not
here." Nobody else is writing it.

`docs/index.md:26-45` — the first thing every new customer copies — is a complete
ROPC example (`authmode`, `username`, `password`, `scopes`). Every attribute in
it is one we are deprecating.

**Assume PM is merged and released with service accounts by the time we cut the
provider release.** So no dual-example hedging:

- Rewrite `templates/index.md.tmpl` so the front-page example is the **secret
  service account**. ROPC is not on the front page at all — just a short
  deprecation note pointing at the guide. (`templates/` is the source;
  `docs/` regenerates via `go generate`.)
- Add one guide, `service-account-authentication.md`: the two modes and how to
  migrate off a password.

**Out of scope:** the per-CI recipes (GitLab, GitHub Actions, Azure DevOps,
Kubernetes). Those are hop-1 instructions, Entra-side rather than ours, and they
need real customer testing before we publish them as working.

## A1 does not depend on PM merging

The provider only ever talks to Keycloak's token endpoint and sends a realm
header. Both exchanges are plain RFC 6749 and RFC 7523 — that contract belongs to
Keycloak, not to MR !1911. We can build and ship A1 whether or not PM lands.
Only the acceptance tests need a real service account, and they stay on ROPC
for now.

## Resulting provider schema

| Attribute | State after A1 |
|---|---|
| `apiurl` | unchanged, still required (A2 makes it optional) |
| `realm` | unchanged (A2 replaces it with `tenant_short_name`) |
| `authurl` | unchanged, still required (A2 makes it optional) |
| `client_id` | **new** — the OAuth client id; secret mode and ROPC |
| `client_secret` | **new**, sensitive — selects secret mode |
| `oidc_token` | **new**, sensitive — the *assertion*, selects federated mode |
| `oidc_token_file` | **new** — path to the assertion, re-read on refresh |
| `clientid` | **deprecated** in favour of `client_id` |
| `username` / `password` | **deprecated** with ROPC |
| `scopes` | **deprecated** — neither service-account mode sends a scope |
| `audience` | **deprecated**, kept and inert — removing it would fail at parse time |
| `authmode` | **deprecated**; `authmode = "auth0"` becomes a clear error |

## Still to do beyond A1

- **A2** — `tenant_short_name`, optional `apiurl`/`authurl` with Axual Cloud defaults.
- **C** — typed member URIs in `axual_group`; `axual_user` and the user data
  source no longer seeing service accounts. Plus moving the acceptance-test
  harness off ROPC onto a service account.
- **B** — the `axual_service_account` resource.
- **TLS (c)** — verify certificates by default, with an opt-out. Breaking for
  on-prem self-signed certs; needs its own deprecation cycle.
- **Report to the design author** — the `oidc_token = "${AXUAL_TOKEN}"` example
  in the upstream design is not valid HCL.
