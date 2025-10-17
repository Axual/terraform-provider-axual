# Developer Guide

This guide is for developers who want to contribute to the Terraform Provider for Axual Platform or build it locally for development purposes.

## Prerequisites

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
  - Recommended: `brew install terraform`
- [Go](https://golang.org/dl/) >= 1.21
  - Recommended: `brew install go`
- Access to an Axual Platform instance for testing (local deployment or Axual Cloud)

## Development Setup

### 1. Configure Local Provider Override

Create or edit the file `~/.terraformrc` to use your locally built provider instead of the one from the registry:

```hcl
provider_installation {
  dev_overrides {
    "axual.com/hackz/axual" = "/Users/<your-username>/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

Replace `<your-username>` with your actual username.

### 2. Install Dependencies

Navigate to the `terraform-provider-axual` directory and download dependencies:

```bash
go mod tidy
```

### 3. Build and Install the Provider

Build and install the provider to your local `$GOPATH/bin`:

```bash
go install
```

Alternatively:

```bash
go build -o $GOPATH/bin/
```

### 4. Test Your Local Provider

Navigate to `terraform-provider-axual/examples/axual` and configure the `provider.tf` file with your credentials. With the dev override configured in `~/.terraformrc`, you can now run:

```bash
terraform plan
terraform apply
```

**Note:** You don't need to run `terraform init` when using dev overrides.

### Troubleshooting Development Setup

If you encounter issues with Go not being found or the provider not being recognized, ensure your environment variables are set correctly in `~/.zshrc` (or `~/.bashrc`):

```bash
export GOPATH=$HOME/go
export GOROOT="$(brew --prefix golang)/libexec"
export PATH="$PATH:${GOPATH}/bin:${GOROOT}/bin"
```

## Debugging

### Debugging in IntelliJ IDEA using Delve

There are four IntelliJ IDEA run configurations available in the `.run` folder.

#### Steps:

1. Install Delve:
   ```bash
   brew install delve
   ```

2. Run **"Axual build to _bin"** to generate the executable in `/go/bin` named `terraform-provider-axual`

3. Run **"Axual Delve binary"** (Run icon) to start Delve's debugging session on this binary

4. Run **"Axual Go Remote"** (Debug icon) to connect IntelliJ to the Delve debugging session. You'll see output like:
   ```
   TF_REATTACH_PROVIDERS='{"axual...
   ```

5. Copy the entire `TF_REATTACH_PROVIDERS` string and export it as an environment variable:
   ```bash
   export TF_REATTACH_PROVIDERS='{"axual...'
   ```
   **Note:** On macOS, this only works in a separate terminal session. Run terraform commands from the same iTerm terminal session.

6. Set a breakpoint in your code and run a terraform command (e.g., `terraform plan` or `terraform apply`). Execution will stop at your breakpoint.

7. When done debugging:
   ```bash
   unset TF_REATTACH_PROVIDERS
   ```

**Alternative:** To debug without generating the binary first, run **"Axual Delve project"** (Run icon).

### Debugging the Webclient Module

The `axual-webclient-exec` module is available for testing and debugging the `axual-webclient` module independently of Terraform functionality.

## Logging

### Logging in axual-webclient

Use Go's standard `log` package:

```go
log.Println("strings.NewReader(string(marshal))", strings.NewReader(string(marshal)))
```

### Logging in the internal folder

Use Terraform's logging framework:

```go
tflog.Error(ctx, "MARSHAL", map[string]interface{}{
  "MARSHAL": string(marshal),
})
```

Then run Terraform with the appropriate log level:

```bash
TF_LOG=ERROR terraform apply -auto-approve
```

Available log levels: `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`

**Note:** Logging only works when using a locally compiled provider, not when using a provider from the registry.

## Documentation

### Generating Documentation

To generate Terraform provider documentation from schema definitions and templates:

```bash
go generate
```

This command generates documentation based on:
- Templates in the `templates/` directory
- `MarkdownDescription` fields in resource schemas

## Terraform Manifest File

The `terraform-registry-manifest.json` file contains:
- **version**: Numeric version of the manifest format (not the provider version)
- **protocol_versions**: Set to `6.0` because this provider uses the Terraform Plugin Framework

## Acceptance Tests

### Prerequisites

Before running acceptance tests:

1. **Configure Tenant:**
   - Update your Tenant's `SupportedAuthenticationMethods` to enable:
     - SSL
     - SCRAM_SHA_512
     - OAUTHBEARER
   - Set the `Update and Deploy Owned Resources` settings to **All Group Members**
   - Set the `Enable Schema Roles for your users` settings to **Disabled**

2. **Configure Test Config:**

   In [`internal/tests/test_config.yaml`](./internal/tests/test_config.yaml), update:
   - `instanceName` and `instanceShortName` to point to an instance with:
     - `EnabledAuthenticationMethod`: SSL, SCRAM_SHA_512, OAUTHBEARER
       - SSL using [Axual Dummy Root CA as the Signing Authority](https://gitlab.com/axual/qa/local-development/-/blob/main/governance/files/axual-dummy-intermediate)
     - `GranularBrowsePermission` : enabled
     - `ConnectSupport`: enabled
     - has an `Apicurio` Schema Registry configured
   - `groupName`: A group you are a member of
   - `userEmail`: Your email
   - `username`: Your username
   - `password`: Your password

3. **Verify Test User Permissions:**

   Ensure your test user has these roles:
   - Tenant Admin (needed for creating Groups and Users)
   - Application Author
   - Environment Author
   - Schema Author
   - Schema Admin (needed for deleting Schemas not assigned to your Group)
   - Topic Author

4. **Configure Provider Connection:**

   Edit [`internal/tests/test_provider.go`](./internal/tests/test_provider.go) to connect to your target environment.

### Running Tests with IntelliJ IDEA

#### Run All Tests

Execute the [`.run/Run all the tests.run.xml`](.run/Run%20all%20the%20tests.run.xml) configuration.

**Environment Variables:**
- `TF_ACC=1` - Built-in safety variable to prevent accidentally running tests on a live environment
- `TF_ACC_TERRAFORM_PATH` - Path to Terraform binary (e.g., `/opt/homebrew/bin/terraform`)
- `TF_LOG=INFO` - Optional but highly recommended for debugging

**Go Tool Arguments:**
- `-p 1` - Disable parallelization to prevent conflicts when creating shared resources
- `-count 1` - Disable test caching to allow running the same tests multiple times

#### Run a Single Test

1. Click the test icon in the IntelliJ IDEA gutter (left of the test function)
2. Choose `Modify Run Configurations`
3. Add environment variables:
   ```
   TF_ACC=1;TF_ACC_TERRAFORM_PATH=/opt/homebrew/bin/terraform;TF_LOG=INFO
   ```
4. Apply → Run test

#### Recommended Testing Order

When running tests for the first time, try them in this order to verify your setup:
1. `user_resource_test.go`
2. `topic_data_source_test.go`
3. `application_deployment_resource_test.go`
4. All tests together

**Note:** If a test fails, you may need to manually delete resources using the Axual Platform UI.

### Running Tests with VS Code or Command Line

#### Option 1: Using Environment File

1. Create a `local.env` file in your project root:
   ```bash
   AXUAL_PASSWORD=<INSERT_PASSWORD>
   AXUAL_USERNAME=<INSERT_USERNAME>
   TF_ACC=1
   TF_ACC_TERRAFORM_PATH=/opt/homebrew/bin/terraform
   TF_LOG=INFO
   ```

2. Run tests:
   ```bash
   go test -p 1 -count 1 ./internal/tests/...
   ```

#### Option 2: Inline Environment Variables

Run tests with environment variables set inline:

```bash
AXUAL_PASSWORD='your_password' \
AXUAL_USERNAME='your_username' \
TF_ACC=1 \
TF_ACC_TERRAFORM_PATH='/opt/homebrew/bin/terraform' \
TF_LOG='INFO' \
go test -p 1 -count 1 ./internal/tests/...
```

### Connecting to Different Environments

To run tests against different Axual Platform instances (e.g., Axual Cloud), modify the provider block in [`test_provider.go`](./internal/tests/test_provider.go):

**Example for Axual Cloud:**

```terraform
provider "axual" {
  apiurl   = "https://axual.cloud/api"
  realm    = "<your-realm-name>"
  username = "<your-username>"
  password = "<your-password>"
  clientid = "self-service"
  authurl  = "https://axual.cloud/auth/realms/<your-realm-name>/protocol/openid-connect/token"
  scopes   = ["openid", "profile", "email"]
}
```

## Writing Tests

### Test Coverage Requirements

When writing acceptance tests, ensure you test:

1. **Creating the resource**
2. **Updating every field:**
   - Optional fields should be able to be removed
   - Sets and Maps:
     - Can be empty
     - Can contain more than one element
3. **Importing** the resource
4. **Deletion** (automatic by the acceptance test framework)

### Preventing Automatic Resource Deletion

To prevent a test from destroying a resource (useful for debugging):

```terraform
resource "axual_group" "team-integrations" {
  name          = "testgroup9999"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members       = [
    "18ac7e79ce4d4063b53787d969742ddd",
  ]
  lifecycle {
    prevent_destroy = true
  }
}
```

**Note:** The test will fail, but this is useful for inspecting what the test actually created in the Axual Platform API.

### Test Logging

Enable logging during test execution:

1. Edit the Go Test run configuration
2. Add environment variable: `TF_LOG=INFO`
3. Add logging statements in your code:
   ```go
   tflog.Info(ctx, fmt.Sprintf("delete group successful for group: %q", data.Id.ValueString()))
   ```

**Note:** Logging only works with a locally compiled provider, not when using a provider from the registry.

### Test Debugging

To debug acceptance tests in IntelliJ IDEA:

1. Set a breakpoint in the resource file (e.g., `resource_group.go`)
2. Add username and password environment variables to the run configuration or temporarily hardcode them
3. Click the Debug button

**Note:** Debugging only works with a locally compiled provider, not when using a provider from the registry.

## Release Process

1. Update the `CHANGELOG.md` version to the target release
2. Update any provider.tf version to the target release in the `examples/`
3. Update `templates/index.md.tmpl` version to the target release
4. Generate the documentation with `go generate` command
5. Git Commit these files
6. Git tag and push
7. Update the https://github.com/Axual/terraform-provider-axual/releases page with the newly released tag
