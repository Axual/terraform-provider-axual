# Domain glossary

Terms this repository uses in a specific way. Implementation details belong in
the code and in `notes/`, not here.

## Identity and authentication

**Tenant** — an Axual customer. Each tenant has a **short name**, which is also
the name of its Keycloak realm. The short name is what the `realm` HTTP header
carries on every API call.

**Realm header** — the HTTP header the provider sends alongside the bearer token,
whose value is the tenant short name. The API Gateway reads it and translates it
into the `X-User-Tenant-Short-Name` header that Platform Manager sees. Platform
Manager never reads the realm header itself.

**ROPC** — Resource Owner Password Credentials, the OAuth2 password grant. The
provider's original and now deprecated way of authenticating: a *human's*
username and password, used by automation. Being removed; OAuth 2.1 drops the
grant entirely.

**Service account (SA)** — a machine identity. In Platform Manager it is a `user`
row with `type = 'SA'`, so it carries the same roles and group memberships a
person would. In Keycloak it is a confidential client in the tenant's own realm.
It is never a person: nothing may mail it, and nothing may render it as one.

**Credential mode** — how a service account proves who it is. Exactly two:
**secret mode** and **federated mode**. Chosen when the service account is
created and not changed afterwards.

**Secret mode** — the service account authenticates with a client id and a client
secret. The secret is shown once at creation and once on each rotation, and is
never stored by Platform Manager.

**Federated mode** — the service account authenticates with no secret at all. It
presents an **assertion** instead, and Keycloak resolves which client is calling
from the assertion's issuer and subject. Nothing to rotate, because Platform
Manager owns no credential.

## Federated mode in detail

**Assertion** — the signed token, issued by the *customer's* identity provider
(in practice Entra), that a federated service account presents in place of a
secret. **It is not the Axual API token.** The provider's `oidc_token` and
`oidc_token_file` attributes hold the assertion; the provider exchanges it for
the API token. Confusing the two is the single easiest mistake to make here.

**Hop 1** — exchanging a runtime's own identity token (a GitLab CI token, a
GitHub Actions token, a Kubernetes projected token) for an Entra token. **The
customer does this**, as a pipeline step, before Terraform runs. The provider has
no part in it. On Azure compute there is no hop 1: the managed-identity token is
already an Entra token.

**Hop 2** — exchanging the assertion for a realm-local Keycloak token, which is
the actual API bearer. **The provider does this**, and re-does it whenever the
token expires. The customer never writes it.

**Federated subject** — the external workload's object id, bound to the service
account's Keycloak client. An assertion can only authenticate the service account
its subject is bound to. Two service accounts may not share one subject.

## Provider

**Auth mode** — which of the three credential kinds the provider resolved for
this run: secret, federated, or ROPC. Inferred from which credentials are
present, never configured directly. Not to be confused with the deprecated
`authmode` attribute, which only ever meant "Keycloak or Auth0" and carried no
information once Auth0 was dropped.
