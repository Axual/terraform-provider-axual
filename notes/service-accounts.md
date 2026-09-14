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

**Start here for the next session.** Full brief at the end of this file, under
"C — brief for a fresh session".

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


---

# C — brief for a fresh session

Written 2026-09-13, at the end of the session that delivered **A1**. A1 is
committed on branch `feat/service-account-authentication` (unpushed at the time
of writing). Everything above this line is A1; everything below is C.

## Why this one has a deadline

Every other piece of work here is on our schedule. C is on Platform Manager's.
The day their MR lands and ships, existing customer configurations start failing
with no change from us. Check whether `axual/platform/mgmt-api!1911` has merged
before planning anything else.

## What PM changes, precisely

From the upstream design's *REST surface (tenant-admin)* section:

1. **`/users` stops serving service accounts.** All three of `GET /api/users/{uid}`
   (**404** for an SA uid), the `/users` collection, and
   `/users/search/findByAttributes` exclude `type = SA`. `/api/service-accounts/{uid}`
   becomes the only detail resource for an SA.

2. **Group membership is written by typed URI.** `members`, `managers` and
   `resourceManagers` carry `/service-accounts/{uid}` for a service account row and
   `/users/{uid}` for a human. **A URI that disagrees with the referenced row's type
   is rejected with 400.**

3. `type` is exposed on any *user* representation that names an actor or a member —
   group member and manager rows, and the attribution surfaces saying who created or
   edited something.

## Where we break

`internal/provider/resource_group.go:271`:

```go
for _, member := range memberUIDs {
    fullURL := fmt.Sprintf("%s/users/%v", apiUrl, member)   // <- 400 for an SA
    members = append(members, fullURL)
}
```

`managers` on the line below builds `%s/groups/%v` and is unaffected — managers are
groups, not users. `resourceManagers` does not appear anywhere in the provider
(checked: no match repo-wide), so it is out of scope. **`members` is the only
breakage.** That is the whole blast radius: one loop, one format string.

## The actual design problem

The design says a client is "correct by construction" if it echoes back the `self`
href PM gave it. **We cannot do that**, and this is the crux of C:

- `axual_group.members` is a `types.Set` of **bare uids** (`resource_group.go:42`).
- Reads discard everything but the uid — `groups_data.go` decodes members as
  `[]struct{ Uid string }`, and `resource_group.go:218-228` rebuilds a set of uids.
- So on **create** there is no prior read to echo. The user writes
  `members = [axual_user.foo.id]` and the provider must decide, from a bare uid
  alone, whether to emit `/users/` or `/service-accounts/`.

Options, none obviously right — this is what to grill next time:

| | Approach | Cost |
|---|---|---|
| a | Widen the read model to keep each member's `type`, and use it on update | Does nothing for create, where there is no prior state |
| b | Look each uid up before writing | An extra call per member. Now plausible: the SA search is open to any tenant user (see below), where a tenant-admin-only endpoint would have been unusable — the provider does not run as tenant admin |
| c | Take typed references in HCL (`axual_service_account.x.id` vs `axual_user.y.id`) | Needs **B** to exist for the SA side; changes the schema |
| d | Try `/users/`, retry as `/service-accounts/` on 400 | Works without B and without a lookup, but it is a guess-and-retry |

(c) is the honest model, (d) is the lazy one that ships today, and (b) became
viable on 2026-09-14 when the SA search opened up to all tenant users. Price all
three; (a) alone is not sufficient because create has no prior state.

## The lookup gap — mostly closed (design is out of date here)

The design flags a gap under *Gap — no member search returns SAs* (line 323),
saying `/service-accounts/search/findByAttributes` is "scoped to the SA admin
screens" and calling the fix "an API story of its own, and a prerequisite".

**That is out of date.** Confirmed 2026-09-14: `/service-accounts/search/findByAttributes`
is now open to **any user of the tenant**, not just tenant admins.

This matters more than it sounds, and it is the reason option (b) moves from
impossible to plausible. The provider authenticates as whatever identity the
customer configured — a service account with topic and environment roles, say.
It is **not** normally a tenant admin. So an endpoint gated on tenant-admin is
unusable to us for resolving a member's type, no matter how convenient: the
customer's own credentials would get a 403 doing routine group management.
With the search open to any tenant user, we can actually call it.

**Two things still to confirm before building on it:**

1. **Can it resolve by uid?** The name says *findByAttributes*, and the design says
   it mirrors `/users/search/findByAttributes`. If the attributes are name/email
   only, it does not answer "what type is uid X", which is the question
   `axual_group` actually has. Check the controller
   (`ServiceAccountDynamicSearchController.java:37`) for the parameters it accepts.
2. **Does it emit the `/service-accounts/{uid}` self href?** The design is emphatic
   that a lookup returning a `/users/` href satisfies a picker and still fails the
   write. Since this is the SA resource's own search, it almost certainly does — but
   it is a one-line check and the failure mode is a 400 at apply time.

Note that `GET /api/service-accounts/{uid}` — the detail route — is a different
question and is likely still tenant-admin gated under the `SERVICE_ACCOUNT_*` ABAC
rules. Do not assume the search being open means the detail route is.

## Also in scope for C

Move the acceptance harness off password authentication onto a service account.
A1 deliberately left this behind because PM had not merged. It touches three places
that each emit credentials independently:

- `internal/tests/test_provider.go:117-128` — the HCL provider block (now without
  `authmode`, which A1 removed from the harness)
- `internal/tests/test_provider.go:201-218` — `apiClient()`, a second path that
  bypasses the provider entirely
- `internal/tests/test_config.yaml` — has no client-id field; `"self-service"` is
  hardcoded at both sites above

## What A1 already verified, so C need not re-check it

Against a live e2e stack on 2026-09-13:

- legacy password-grant configs still authenticate (the `AuthStyleInParams` risk)
- a secret-mode service account authenticates and does real work — created an
  environment, a topic and a topic config, and re-planned clean
- credentials resolve from the environment; a stale `AXUAL_AUTH_PASSWORD` is ignored
  with a warning naming it
- every error path, including a federated exchange reaching real Keycloak

Still unverified anywhere: **federated mode with a valid assertion.** PM's e2e suite
carries a stand-in IdP realm called `e2eidp` that could close it.

## Bringing up a stack to test against

In `~/IdeaProjects/gitlab/axual-flux`, on the service-accounts branch:

```bash
cd e2e && ./run.sh up      # ~13 containers; down / down -v to stop
```

Gives `http://localhost:8080/api`, Keycloak at `http://localhost:8070/auth`, tenant
realm `e2e`. Seeded users have password == username (`tenant-admin`, `tina`, ...) and
the OAuth client for the password grant is `self-service`.

**Note:** an instance created through the API without bootstrap servers is not wired
to the running Kafka broker, and every test that touches a topic then fails with a
`500`. Use an instance that is actually wired to a cluster, or expect 11 of the 19
acceptance packages to fail for reasons unrelated to the change under test.

---

# C — decisions

Written 2026-09-14. The brief above is what we believed going in; this section is
what we found and what we built. Where the two disagree, this section is right.

## What the brief got wrong

Two rounds of questions to Platform Manager (`axual-flux`, branch
`feat/axpd-11914-create-service-account`) moved four of the brief's assumptions:

| The brief assumed | Actually true |
|---|---|
| The SA search can resolve a uid | **No.** `findByAttributes` takes `name`, `clientId`, `roles` only (`ServiceAccountDynamicSearch.java:22-26`) |
| There is no cheap type oracle | **There is.** `GET /service-accounts/{uid}` is open to *any* tenant user — the ABAC condition is literally `"true"` (`SimplePolicyDefinition.java:888-889`), proven for a non-admin at `ServiceAccountIT.java:196-211` |
| Reads discard everything but the uid | PM now sends `type` (`"regular"` / `"SA"`) **and** a typed `self` href per member (`GroupUserDTO.java:45-51`, `:98-116`) |
| A 400 identifies a type mismatch | **It does not.** Wrong collection and unknown uid throw the same `InvalidRequestException` with the same text, `"URI does not represent a valid resource"` (`GroupMembershipService.java:48-51`) |

Two the brief did not mention at all:

- **A bare uid still works, and means "a person"** (`GroupMembershipService.java:36`,
  test `mapUsers_should_treatABareUidAsAHuman`). Kept for older fixtures.
- **`/users/search/findByEmailAddress` does *not* exclude service accounts.** It is
  the one query in `UserRepository.java` missing the `type <> SA` clause that every
  sibling has (lines 101-103). Service accounts do have a real, synthesised email,
  `<clientId>@sa.axual.io` (`ServiceAccountIdentity.java:15-16`). So the brief's fear
  that `axual_user` and the user data source would "silently lose" service accounts is
  backwards: the data source silently *includes* them.

## The decision: probe each member (option b), via the detail route

The four options in the brief, repriced:

| | Approach | Verdict |
|---|---|---|
| a | Keep each member's `type` from reads, use it on update | **Insufficient alone.** Create has no prior read to echo |
| **b** | **Ask the API what each uid is, before writing** | **Chosen.** Works identically on create and update |
| c | Typed references in HCL | **Deferred.** Needs **B** to exist, and changes the schema |
| d | Try `/users/`, retry on 400 | **Dead.** The 400 is indistinguishable from an unknown uid, so a typo would trigger a pointless retry and then report the wrong cause |

(b) is not the option the brief described. The brief expected it to run through
`/service-accounts/search/findByAttributes`, which cannot take a uid. It runs through
the **detail route** `GET /service-accounts/{uid}` instead — one call, and open to any
tenant user, which is what matters: the provider authenticates as whatever identity the
customer configured and is **not** normally a tenant admin.

Deliberately **not** done:

- **No caching** of probe results across members or across groups within a run.
  Groups hold a handful of members. Add it when a real configuration makes it hurt.
- **Not combined with (a).** Storing `type` in state would save probes on update only,
  never on create, in exchange for widening the state shape. The probe already behaves
  the same on both paths.
- **No better error on a 400.** After the probe, a 400 on members can only mean the uid
  does not exist — but PM returns the same status for unrelated group validation, so
  explaining it would be a guess. A1's `invalid_client` message worked because the eager
  fetch gave a genuinely free discriminator; there is no equivalent here.

## Would asking PM for a uid parameter have helped?

Only marginally, and only if it took a *list* of uids — that is the one thing the detail
route cannot do, turning N calls into 1. N is small. The ambiguity would not improve:
a uid that is not an SA and a uid that does not exist both come back empty, exactly like
the 404.

It was rejected on scheduling, not on merit. C is the only work item on **Platform
Manager's** clock rather than ours; adding a PM dependency would put it behind a second
merge. If PM later adds batch uid search, swapping the helper's body is a ten-line
change and no call site cares.

The higher-value ask, if one is ever made, is different: **let PM resolve the type from a
bare uid itself.** PM already knows every row's type; the typed-URI design pushes a
lookup onto every client that only PM can answer for free, and bare uid is already
accepted and already means "a person". That reverses a documented decision
(`GroupMembershipService.java:29-36`), so it is a design conversation, not a patch.

## Why the read side needed nothing

Verified against the branch: `GroupUserDTO` still serialises `uid`
(`GroupUserDTO.java:41-43`, asserted at `GroupIT.java:977`), the collection keys are still
`_embedded.members` / `_embedded.managers` (`GroupRepresentations.java:23-24,44-45`), and
a group **write** returns the same representation as a read
(`CustomGroupController.java:94,102,115`). So `mapGroupResponseToData` and the group data
source are untouched.

`resourceManagers` is present but feature-flagged and additive
(`GroupRepresentations.java:25,47-53`), and still appears nowhere in this provider.
Out of scope, as the brief said.

## Blast radius, confirmed

One loop. `managers` builds `%s/groups/%v` and is unaffected — a group's managers are
groups, not people.

## What shipped

- `axual-webclient/groups.go` — `GroupMemberURI(uid)`. 404 means "write this as a
  person", which covers a person, an unknown uid, **and a Platform Manager old enough to
  have no `/service-accounts` route at all**. Any other failure aborts the write rather
  than guessing: falling back to `/users/` would send a person's URI for a service
  account and turn an outage into a 400 naming the member.
- `internal/provider/resource_group.go` — the loop calls it; `createGroupRequestFromData`
  takes the client instead of the API URL (a *smaller* diff, since both call sites were
  already passing `client.ApiURL`).
- `axual-webclient/groups_test.go` — table tests against `httptest`: each kind alone, a
  mixed member list, non-404 failures aborting, and a Platform Manager without service
  accounts.
- `members` description on both the resource and the data source, plus regenerated docs.

Old Platform Manager compatibility is not incidental — it is why no acceptance test
needed changing. A human uid 404s on `/service-accounts/{uid}` against old *and* new
Platform Manager, so every existing group test keeps passing, on ROPC, unchanged.

## Deferred out of C, deliberately

- **The acceptance harness stays on ROPC.** Moving it onto a service account is real work
  in three places (`test_provider.go:117-128`, `apiClient()` at `:201-218`,
  `test_config.yaml`, which also hardcodes `"self-service"` at both sites) and it is not
  what breaks when PM ships. It moves when ROPC is removed.

  When it does move: A1 left the seam for it. `webclient.Resolve` is pure and exported,
  so `apiClient()` can stop hand-building `Credentials{Mode: ModeROPC, …}` and share the
  provider's own resolution instead of duplicating it. There is **no pre-seeded service
  account to reuse** — the e2e baseline accounts never capture their secrets
  (`e2e/tests/service-accounts/conftest.py:37-60`) and PM returns a secret exactly once,
  on create and rotate (`ServiceAccountResult.java:9-13`). A developer creates one by
  hand and pastes the secret in. A service account may hold `TENANT_ADMIN`; only
  `SUPER_ADMIN` and `BILLING_INTERNAL` are refused (`ServiceAccountService.java:480-489`).

- **No acceptance test for a service account member.** It would need the harness on a
  service account, per above. The unit tests cover the URI selection; what they cannot
  cover is PM actually accepting the URI.

## To raise with Platform Manager

1. **`findByEmailAddress` is missing its `type <> SA` filter** (`UserRepository.java:101-103`).
   Every sibling query has it. Looks like an oversight rather than a decision, and it means
   `data "axual_user"` returns a service account for anyone who knows the synthetic email.
   Deliberately **not** worked around client-side: the response carries no `type`, so
   filtering would cost a probe per read to reject a result the user asked for by name.
2. **`type` should probably not be on a public representation at all.**
   `GroupUserDTO.getType()` (`GroupUserDTO.java:49-51`) serialises on every member,
   manager and resourceManager row of every group response, and its value is
   `user.getType().getDbValue()` — the raw database column, `"regular"` or `"SA"`
   (`UserType.java:15-16`).

   It is redundant: the typed `self` href already says the same thing, and the design's
   own "correct by construction" principle makes the href the value clients are meant to
   echo back. And it leaks storage into the contract — that a person and a service
   account share one table with a discriminator column is exactly the detail the separate
   `/users` and `/service-accounts` routes exist to hide.

   There is a real need behind it — the design says nothing may render a service account
   as a person, so the UI must distinguish — but the `self` href answers that. If PM
   wants an explicit field anyway, it should not be `getDbValue()`.

   **This provider does not depend on it either way.** `GroupMemberURI` probes
   `/service-accounts/{uid}` rather than reading `type`, and the field is decoded
   nowhere: group members are still `struct{ Uid string }`, and no resource or data
   source exposes a user type. PM can drop the field without touching us.
3. `CLAUDE.md` in this repo tells contributors the test user needs "Topic Author".
   No such role exists — it is `STREAM_AUTHOR` (`Role.java:19-30`), because topics are
   Streams internally. Ours to fix, noted here so it is not lost.
