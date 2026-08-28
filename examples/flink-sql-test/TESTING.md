# Testing the Flink SQL provider changes (AXPD-11717)

Isolated test bed in this directory — separate `terraform.tfstate`, cannot touch
resources managed by `examples/axual/*.tf`.

`main.tf` is a full example: a fresh environment, a Flink Cluster (managed by this
config — none existed on this instance-cluster), one FLINK_SQL application, its
credential, 4 topics (2 consume: String + Avro, 2 produce: String + Protobuf) with
schemas and topic configs, 4 access grants + approvals, and one deployment whose SQL
reads both consume topics and writes to both produce topics. Everything is created
and destroyed by this config alone — the `dev` instance and its `jupiter` Kafka
cluster are pre-existing shared infrastructure, referenced read-only via data
sources and never touched.

## Prerequisites

- Local provider binary installed and picked up via dev_overrides:
  ```
  cd /Users/cemretok/IdeaProjects/terraform-provider-axual
  
  
  
  ```
  `~/.terraformrc` must have:
  ```
  provider_installation {
    dev_overrides {
      "registry.terraform.io/Axual/axual" = "/Users/cemretok/go/bin"
    }
    direct {}
  }
  ```
- `provider.tf` in this directory is pointed at the dev/QA env:
  `https://self-service.qa.np.westeurope.azure.axual.cloud`, user `admin-axual`.

## Variables to fill in `main.tf`

Already set to real, confirmed-working values on this tenant:

| Variable | Value | Notes |
|---|---|---|
| `instance_short_name` | `dev` | instance uid `e467f0f81276426780711e415d33f33b`; **Flink-enabled** (the `qa` instance is not) |
| `group_name` | `Admins` | owner group used for all created resources |
| `your_email` | `admin@axual.com` | must be a real Self Service user |
| `instance_cluster_id` | `f076e00035ab49a1be62916b5691b569` | the "jupiter" cluster on the `dev` instance |
| `flink_api_token` | *(not committed — see below)* | Ververica namespace-scoped token |

`flink_api_token` is sensitive — don't hardcode it in `main.tf`. Pass it as an
environment variable instead:

```
export TF_VAR_flink_api_token="<your ververica token>"
```

## Commands

```
cd examples/flink-sql-test

# see what would change
terraform plan

# create everything
terraform apply -auto-approve

# tear it all down when done
terraform destroy -auto-approve
```

## ⚠️ Destroy order matters for FLINK_SQL

**Always destroy `axual_application_deployment` before its `axual_flink_cluster`.**
`terraform destroy` normally handles dependency order correctly on its own — this
only bites if you manually run `terraform state rm` on the deployment (or on any
resource it depends on) instead of a real destroy.

Two confirmed backend bugs make a stranded deployment **permanently unrecoverable**
if the order is violated:
- A FLINK_SQL deployment's `target_id` cannot be changed after creation:
  `"Changing the deployment target of a FLINK_SQL deployment is not yet supported."`
- `DELETE` on the deployment requires its original target's Flink Cluster to still
  exist: `"No Flink Deployment Target matching the deployment.targetId"` — and
  Flink Cluster IDs are server-generated, so the same ID can never be recreated.

If you hit this: there is no API-level recovery. The deployment (and anything that
depends on it — its application, environment) is only fixable via DB-level cleanup
by the platform team. `terraform state rm` the affected resources to stop tracking
them and move on; do not keep retrying destroy against them.

**Never manually `terraform state rm` any resource in this config for any reason.**
Just run `terraform apply` / `terraform destroy` and let Terraform's own dependency
graph order the operations — that's what keeps this safe. The stranding above only
happened because a previous version of this test manually removed the deployment
from state and then destroyed its Flink Cluster out from under it.

## Confirmed working end-to-end (2026-08-27)

Full pipeline verified against this environment, `state: "Running"` confirmed live
on the deployment via the API - Terraform → mgmt-api → Ververica, all working:

- `axual_flink_cluster`: Create, Import, Update, Delete
- `axual_application` with `FLINK_SQL`
- `axual_topic`, `axual_topic_config`, `axual_application_access_grant(_approval)`,
  `axual_application_credential`
- `axual_application_deployment`: Create (target-only POST → PATCH config → START),
  Update (STOP → PATCH → START), Delete (STOP → DELETE) — confirmed AXPD-11714 is
  live on this environment as of 2026-08-27.

## Known discrepancies vs. AXPD-11717 confirmed against the real backend

1. **FLINK_SQL apps DO need `axual_application_credential`** before an access grant
   can be created (`400 "Please provide authentication information for the
   application..."` without one) — the ticket claims credentials are injected
   automatically and none is needed. Not true against this backend.
2. ~~`DELETE` unsupported for FLINK_SQL deployments~~ — was true before AXPD-11714
   was deployed to this environment; confirmed fixed as of 2026-08-27. See the
   destroy-order warning above for the one remaining sharp edge.
