---
subcategory: ""
page_title: "Service account authentication"
description: |-
  How to authenticate the Axual provider as a service account, with a client secret or with no secret at all, and how to migrate from username and password authentication.
---

# Service account authentication

A **service account** is a machine identity. It holds roles and group memberships just
like a person does, but nothing about it belongs to an individual: it does not leave when
someone changes team, and everything it does is recorded as machine activity rather than
as a person's.

Before service accounts existed, the only way to run Terraform against Axual was to give
it a real person's username and password. That still works, and emits a deprecation
warning, but it puts a password into every pipeline that needs it and ties your automation
to whoever owns the account. Use a service account instead.

A tenant admin creates service accounts. Ask yours for one, and say which groups and roles
it needs — a service account can only manage what its roles and group memberships allow,
exactly like a person.

## Choosing a mode

There are two ways a service account can prove who it is. Both are OAuth2 client
credentials; the difference is what the provider presents.

| | Client secret | Federated |
|---|---|---|
| What you hold | A secret, shown once at creation | Nothing |
| Where it comes from | Axual | Your own identity provider |
| Rotation | You rotate it, and update the pipeline | Nothing to rotate |
| Best for | Getting started, and anywhere a secret can be stored safely | CI and Kubernetes, and any account with wide permissions |

Prefer **federated** where you can. There is no secret to leak, copy into a second place,
or forget to rotate.

The provider works out which mode to use from the credentials you supply. You never choose
it directly.

## Client secret

```hcl
provider "axual" {
  apiurl  = "https://axual.cloud/api"
  realm   = "acme"
  authurl = "https://axual.cloud/auth/realms/acme/protocol/openid-connect/token"

  client_id     = "sa-orders-producer-a1b2c3"
  client_secret = var.axual_client_secret
}
```

Every credential can come from the environment instead, which is usually what you want in
a pipeline — a secret in a `.tf` file ends up in version control.

| Attribute | Environment variable |
|---|---|
| `client_id` | `AXUAL_CLIENT_ID` |
| `client_secret` | `AXUAL_CLIENT_SECRET` |
| `oidc_token` | `AXUAL_OIDC_TOKEN` |
| `oidc_token_file` | `AXUAL_OIDC_TOKEN_FILE` |

With both set in the environment, the provider block needs no credentials at all:

```hcl
provider "axual" {
  apiurl  = "https://axual.cloud/api"
  realm   = "acme"
  authurl = "https://axual.cloud/auth/realms/acme/protocol/openid-connect/token"
}
```

The secret is shown once, when the account is created and again each time it is rotated.
Axual does not store it and cannot show it to you later. Rotation replaces the old secret
immediately, so update your pipeline first.

## Federated

A federated service account holds no secret. Instead it presents an **assertion**: a token
issued by your own identity provider that proves which workload is calling. Axual is
configured to trust that provider and to accept assertions for one specific workload.

The assertion is **not** an Axual token. The provider exchanges it for one.

```hcl
provider "axual" {
  apiurl  = "https://axual.cloud/api"
  realm   = "acme"
  authurl = "https://axual.cloud/auth/realms/acme/protocol/openid-connect/token"

  # the assertion is read from AXUAL_OIDC_TOKEN
}
```

~> Do not write `oidc_token = "${AXUAL_OIDC_TOKEN}"`. Terraform has no shell-environment
interpolation, so that is a reference to a Terraform object of that name and fails with
"Reference to undeclared resource". Either leave the attribute out and let the provider
read the environment variable, or use `oidc_token = var.axual_token` with
`TF_VAR_axual_token`.

### Getting the assertion

Producing the assertion is your pipeline's job, not the provider's — how a workload proves
its identity depends entirely on where it runs. The provider only transports it.

Your tenant admin will tell you which identity provider Axual trusts and which audience
the assertion must carry. In most cases the workload first exchanges its own CI or cluster
token for one from that provider, as a normal pipeline step, and the result is what the
provider uses.

In GitLab CI, for example:

```yaml
deploy:
  id_tokens:
    GITLAB_OIDC: { aud: "api://AzureADTokenExchange" }
  script:
    # Exchange the GitLab token for one from the identity provider Axual trusts,
    # and export it under the name the provider already reads.
    - export AXUAL_OIDC_TOKEN=$(...)
    - terraform apply -auto-approve
```

### `oidc_token` or `oidc_token_file`

Use `oidc_token_file` whenever something rewrites the assertion as it rotates — Kubernetes
projected service account tokens are the common case. The provider reads the file again
every time it renews its access token, so a rotated assertion is picked up automatically.

```hcl
provider "axual" {
  # ...
  oidc_token_file = "/var/run/secrets/axual/token"
}
```

An assertion supplied in `oidc_token` is read once and cannot be renewed. Axual access
tokens are short-lived, so a long `terraform apply` can outlive the assertion and fail
partway through. If that happens the provider says so explicitly. Either use
`oidc_token_file`, or ask your platform administrator to allow assertions to live long
enough to cover the whole run.

## Migrating from username and password

1. Ask a tenant admin for a service account with the groups and roles your configuration
   needs. Start by matching what the person whose credentials you are replacing had.
2. Put its client ID and secret where your pipeline keeps secrets, as `AXUAL_CLIENT_ID`
   and `AXUAL_CLIENT_SECRET`.
3. Remove `username`, `password` and `scopes` from the provider block. Remove `authmode`
   and `audience` too if they are there; they no longer do anything.
4. Run `terraform plan`. It should report no changes — you have changed who is calling, not
   what is being managed.
5. Remove `AXUAL_AUTH_USERNAME` and `AXUAL_AUTH_PASSWORD` from your pipeline.

You can do step 5 whenever you like. If a service account credential is present the
provider uses it and ignores any leftover username and password, telling you what it
ignored. A stale environment variable will not break your pipeline.

If the plan in step 4 is **not** empty, the service account is missing a group membership
or a role that the person had. Ask your tenant admin to compare the two.

## What changed, and what still works

Everything below still works and now emits a deprecation warning. Nothing has been
removed, so no existing configuration fails to load.

| | |
|---|---|
| `username` / `password` | Still work. Use a service account instead. |
| `clientid` | Renamed to `client_id`. Both work; setting both to different values is an error. |
| `scopes` | Only applies to username and password authentication. Service accounts do not send a scope. |
| `authmode` | No longer has any effect. Remove it. |
| `audience` | No longer has any effect. Remove it. |
