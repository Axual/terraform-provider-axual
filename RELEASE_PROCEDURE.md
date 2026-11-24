# Release Procedure

This file has every step of a release of the Axual Terraform provider: the version, the changes, the tests,
the tag and what comes after. Follow it from top to bottom.

## Overview

- `master` is the development branch. Every pull request targets `master`.
- `release/vX.Y.x` branches (for example `release/v3.2.x`) exist only for patch releases of an older minor version.
- A release is a tag `vX.Y.Z`. Pushing the tag starts the GitHub workflow `Release` (`.github/workflows/release.yml`).
  It runs GoReleaser (`.goreleaser.yml`): it builds the provider for every platform, signs the checksums with the
  project GPG key, and creates the GitHub release with the files. The Terraform Registry picks up the new release
  from GitHub by itself.
- GoReleaser writes no release notes (`changelog: disable: true`). The release notes are written by hand (Step 7),
  from `CHANGELOG.md`.
- CI (`.github/workflows/test.yml`) runs the unit tests and checks the docs. It runs no acceptance tests, so the
  acceptance tests in Step 4 are the only test of the provider against a real Platform Manager.

## Version numbers

| Change | Version | Example |
| --- | --- | --- |
| A breaking change: an existing configuration fails or changes behaviour after the upgrade | Major | 3.2.0 to 4.0.0 |
| New resources, attributes or features that keep existing configurations working | Minor | 3.2.0 to 3.3.0 |
| Bug fixes only | Patch | 3.2.0 to 3.2.1 |

Many users pin the provider with `~>` (for example `version = "~> 3.2"`), so a minor release reaches them with
`terraform init -upgrade`. When the team decides to ship a breaking change in a minor release anyway (as in
3.3.0), say so at the top of the version's section in `CHANGELOG.md` and at the top of the release notes.

## Step 1: Check that master is ready

1. All pull requests for the release are merged into `master`. Check the open pull requests, also Dependabot.
2. The last `Test` workflow run on `master` is green (GitHub, Actions).
3. `CHANGELOG.md` has a line for every user-visible change since the last release. Compare with
   `git log --oneline v<last release>..origin/master`.

## Step 2: Choose the version

Read the `[Unreleased]` section of `CHANGELOG.md`. A line marked **Breaking** normally means a major release (see
"Version numbers", also for the exception). Agree the version with the team before you go on.

## Step 3: Prepare the release changes

Work on a branch from the newest `master`:

```bash
git fetch origin
git switch -c prepare-release-vX.Y.Z origin/master
```

| File | Change |
| --- | --- |
| `CHANGELOG.md` | `## [Unreleased]` becomes `## [X.Y.Z](https://github.com/Axual/terraform-provider-axual/releases/tag/vX.Y.Z) - YYYY-MM-DD`. Write `TBR` as the date until the release day. For breaking changes in a minor release, add the warning (see "Version numbers") |
| `README.md`, `templates/index.md.tmpl`, `templates/guides/*.md`, `examples/*/provider.tf` | Check the `version` pins. A pin that says "the first release with ..." must name a released version, or this one. A pin to an exact version (`examples/terraform-query`) becomes `X.Y.Z` |
| `templates/index.md.tmpl` | Add a row for `X.Y.x` to the Compatibility table, with the Platform Manager and platform versions it supports |
| `docs/` | Generated: run `go generate ./...` (see below). Never edit `docs/` by hand |

Generate the docs:

```bash
go generate ./...
git status
```

- `go generate` must run in a folder named `terraform-provider-axual`. If your checkout has another name, copy it
  to such a folder, run it there, and copy `docs/` back.
- It uses the first `terraform` on `PATH`. Use Terraform 1.14 or newer: an older one knows no list resources,
  deletes every page in `docs/list-resources/` and changes other pages, with exit code 0.
- `git status` must show only the files you expect.

## Step 4: Test the release

Test the commit that you will tag: `master` plus the release changes of Step 3.

### 4.1 Local checks

The `Test` workflow runs on the release pull request. Run the same steps locally first, plus `go mod tidy`: the
`Release` workflow runs `go mod tidy` before it builds, and GoReleaser stops when that changes a file.

```bash
go build ./...
go vet ./...
go test -count 1 ./internal/provider/
GOARCH=386 go test -count 1 ./internal/provider/      # 32-bit (386 and arm are released); Linux only, CI runs it
(cd axual-webclient && go test -count 1 ./...)        # the module with its own dependency versions
go test -count 1 axual-webclient/...                  # from the repository root: the versions the provider ships
go mod tidy && git diff --exit-code go.mod go.sum
```

### 4.2 Release build

CI never runs GoReleaser, so build all platforms once before the tag:

```bash
git archive HEAD | tar -x -C /path/to/empty/dir && cd /path/to/empty/dir
git init -q && git add -A && git commit -qm snapshot
goreleaser build --snapshot --clean
ls dist
```

- Use GoReleaser 2.x, the same major version as the `Release` workflow. An old GoReleaser can fail on targets
  that the current Go version dropped.
- Expect one binary per platform in `.goreleaser.yml` (`goos` times `goarch`, minus the `ignore` list and the
  pairs Go does not support): 13 for 3.3.0, the same platforms as the release before. Compare with the files of the
  last GitHub release.

### 4.3 Acceptance tests on three Platform Manager versions

Run all acceptance test packages (`internal/tests/*`) against:

| Platform Manager | Why |
| --- | --- |
| The newest `master` build | Catches changes in the next Platform Manager release |
| The newest release of the current minor line (for example 16.0.x) | What most customers run |
| The newest release of the long-term support line (for example 15.0.x) | The oldest line that customers still get fixes for |

How:

1. Set up a Platform Manager of that version with Kafka, a schema registry, Kafka Connect and KSML, for example
   [Axual Desktop](https://gitlab.com/axual/public/axual-desktop) with its Platform Manager image changed to the
   exact release.
2. Fill in `internal/tests/test_config.yaml` for that Platform Manager (see "Acceptance Tests" in
   `DEVELOPERS.md`). Set `iamGroupTests: true` and `strict: true`, and `flinkTests: false` where there is no
   Ververica.
3. Run the packages one at a time, and delete leftovers between packages. A failed test leaves its resources
   behind, and the next package that uses the same names then fails too.

   ```bash
   for pkg in internal/tests/*/; do
     TF_ACC=1 TF_ACC_TERRAFORM_PATH=$(command -v terraform) go test -count 1 -timeout 1800s "./$pkg..."
   done
   ```

   Use Terraform 1.14 or newer: the `terraform query` tests need it.

   A package whose tests all skip still reports `ok`. With `strict: true`, a test whose `test_config.yaml` value
   is missing fails instead of skipping. Run with `-v` and read the `--- SKIP` lines: only tests for a feature of
   a newer Platform Manager, and the Flink tests with `flinkTests: false`, may skip.
4. The Flink tests (`FlinkClusterResource` and the `FLINK_SQL` tests) need a Platform Manager with Ververica.
   Axual runs them on its QA environment.
5. For every failure, decide: a provider bug (fix it before the release), a feature that needs a newer Platform
   Manager (the `CHANGELOG.md` line says "needs Platform Manager X or later"; check that the error says so too), or
   a problem of the test environment. Write the results in the release pull request.

### 4.4 Upgrade check

Apply a configuration with the last released provider, then run `terraform plan` with the new build (a
`dev_overrides` CLI configuration, see `DEVELOPERS.md`). The plan must show only the changes that `CHANGELOG.md`
announces for the upgrade, and the next plan after an apply must be empty.

1. In an empty folder, pin the last release (`version = "X.Y.Z"`), run `terraform init` and `terraform apply`
   with the Terraform Registry (no `dev_overrides`). Use the resources most users have: environment, topic,
   topic configuration, application, principal, credential, access grant and its approval or decision.
2. Run `terraform plan` with the new build. Compare every planned change with `CHANGELOG.md`.
3. Apply it. A change that `CHANGELOG.md` calls "no API call" must not change anything in Platform Manager.
4. Try every upgrade path that `CHANGELOG.md` promises, for example a `moved` block from a deprecated resource to
   its replacement: the plan must say "has moved" with 0 to add, change or destroy.
5. `terraform destroy` with the new build. When the environment delete fails once with `403` while the
   application is deleted at the same time, run it again: a known Platform Manager race, not a provider problem.

## Step 5: The release pull request

1. Commit the changes of Step 3 with the message `Release X.Y.Z` and open a pull request to `master`. Put the
   test results of Step 4 in the description.
2. On the release day, set the date in `CHANGELOG.md` (replace `TBR`).
3. Get a review and squash-merge it.

## Step 6: Tag and publish

1. Tag the merge commit on `master` and push the tag:

   ```bash
   git fetch origin
   git log -1 --oneline origin/master        # must be the merge commit of the release pull request
   git tag -a vX.Y.Z -m "Release X.Y.Z" origin/master
   git push origin vX.Y.Z
   ```

2. Watch the `Release` workflow (GitHub, Actions). It takes a few minutes.
3. Check the GitHub release: a zip per platform, `terraform-provider-axual_X.Y.Z_SHA256SUMS`, its `.sig` file and
   `terraform-provider-axual_X.Y.Z_manifest.json`.
4. Check that the Terraform Registry shows the version (https://registry.terraform.io/providers/Axual/axual/latest).
   This can take some minutes.
5. Try it: `terraform init -upgrade` with `version = "X.Y.Z"` in an empty folder.

## Step 7: Release notes

GoReleaser creates the GitHub release without notes. Edit the release on GitHub:

- Start with a warning when the release has breaking changes, with what users must change before they upgrade.
- Then a short "Highlights" section: the main new features, a few sentences each.
- End with a link to the version's section in `CHANGELOG.md`.

## Step 8: After the release

- Close the GitHub issues that the release fixes, with a comment that names the version.
- Axual internal: release the Jira fix version. Rename `terraform-provider-next` to `terraform-provider-vX.Y.Z`,
  create a new `terraform-provider-next`, and release the renamed version with the tag date.
- For a minor or major release, create the branch `release/vX.Y.x` from the tag when a patch release is needed
  later: `git push origin vX.Y.Z:refs/heads/release/vX.Y.x`.

## Patch release from a release branch

1. Cherry-pick the fix onto `release/vX.Y.x` in a pull request to that branch, with a `CHANGELOG.md` line.
2. Run Step 4 on that branch, with at least the packages the fix touches.
3. Tag the merge commit on the release branch (`vX.Y.Z+1`) and continue with Step 6.
4. Make sure `master` has the same fix and the `CHANGELOG.md` line.

## If something goes wrong

- The `Release` workflow failed before it created the GitHub release: delete the tag
  (`git push origin :refs/tags/vX.Y.Z` and `git tag -d vX.Y.Z`), fix the problem in a pull request, and tag again.
- The release is published and broken: do not delete or move the tag, because the Terraform Registry and users
  may already have it. Fix the problem and publish a patch release.

## Checklist

- [ ] Master ready: pull requests merged, `Test` green, `CHANGELOG.md` complete (Step 1)
- [ ] Version agreed (Step 2)
- [ ] `CHANGELOG.md`, version pins, `go generate` (Step 3)
- [ ] Local checks and release build (4.1, 4.2)
- [ ] Acceptance tests on Platform Manager master, the current minor line and the LTS line, and Flink (4.3)
- [ ] Upgrade check (4.4)
- [ ] Release pull request merged, with the date set (Step 5)
- [ ] Tag pushed, workflow green, files and Registry checked (Step 6)
- [ ] Release notes (Step 7)
- [ ] Issues closed, Jira version released (Step 8)
