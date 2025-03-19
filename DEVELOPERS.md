## Acceptance tests
### How to run tests with IntelliJ IDEA
To run the acceptance tests against a running Axual Platform,
you would need to perform some actions.

- Update your Tenant `SupportedAuthenticationMethods` to have these options enabled
    - SSL
    - SCRAM_SHA_512
- In the [`test_config.yaml`](./internal/tests/test_config.yaml), replace the `instanceName` to an available Instance which has the following properties
    - `EnabledAuthenticationMethod`
        - SSL (use the [Axual Dummy Root CA as the Signing Authority](TODO lik))
        - SCRAM_SHA_512
    - `GranularBrowsePermission` enabled
    - `ConnectSupport` enabled
- Then in the [`test_config.yaml`](./internal/tests/test_config.yaml), replace the `groupName` to a Group you are a member of.
- Edit this run configuration included in this repo: [`.run/Run all the tests.run.xml`](.run/Run%20all%20the%20tests.run.xml)
    - Open the edit configuration
    - Look at the env variables section and update the following variables
        - AXUAL_USERNAME=<your username to authenticate with the Platform Manager>
        - AXUAL_PASSWORD=<your password to authenticate with the Platform Manager>
        - TF_ACC=1
            - Built in safety env var for accidentally running the tests on a live environment
        - TF_ACC_TERRAFORM_PATH
            - path to Terraform binary in your local system
        - TF_LOG=INFO
            - Optional but highly recommended
  > Here is a full env variables example: `AXUAL_PASSWORD=<INSERT API PASSWORD>;AXUAL_USERNAME=<INSERT API USERNAME>;TF_ACC=1;TF_ACC_TERRAFORM_PATH=/opt/homebrew/bin/terraform;TF_LOG=INFO`
  - Make sure the test user match the following:
    - the test user has the "Tenant Admin" role, this is needed for creating Groups and Users.
    - the test user is marked as "Resource Manager" on the test group, this is needed for updating any owned resource.
- Make sure to turn off parallelization for running go tests because of conflicts when creating shared resources many times
    - Use this go tool argument: `-p 1`
- Make sure to turn off test caching, because then we can run the same tests multiple times to test stability without having to change the test.
    - Use this go tool argument: `-count 1`


Now you are ready to run the Acceptance Tests.

- First try to run one acceptance test, before trying to run all the tests. It might happen that if a test fails, you have to manually delete resources using the UI.
    - We recommend trying to run in this order:
        - user_resource_test.go
        - topic_data_source_test.go
        - application_deployment_resource_test.go
        - All the tests together

- To run only one test in IntelliJ IDEA:
    - Click on test icon in IntelliJ IDEA for a test file like `user_resource_test.go`(left of func, in gutter)
    - Choose `Modify Run Configurations`
    - Paste the full env variables:
        - For example: `AXUAL_PASSWORD=<INSERT API PASSWORD>;AXUAL_USERNAME=<INSERT API USERNAME>;TF_ACC=1;TF_ACC_TERRAFORM_PATH=/opt/homebrew/bin/terraform;TF_LOG=INFO`
    - Apply -> Run test

### How to run tests in VS Code or command line

- Ensure you have Go installed and set the GOPATH in your system environment variables.
- Add the Go extension to your VS Code
- Create a local.env file in your project root to store your environment variables:
    - AXUAL_PASSWORD=<INSERT API PASSWORD>;AXUAL_USERNAME=<INSERT API USERNAME>;TF_ACC=1;TF_ACC_TERRAFORM_PATH=/opt/homebrew/bin/terraform;TF_LOG=INFO
- Run `go test -p 1 -count 1 ./internal/tests/…`  to run the tests
- You can also run `AXUAL_PASSWORD='your_password' AXUAL_USERNAME='your_username' TF_ACC=1 TF_ACC_TERRAFORM_PATH='/opt/homebrew/bin/terraform' TF_LOG='INFO' go test -p 1 -count 1 ./internal/tests/..` if you don't want to create a local.env file.

### How to connect to a different API
- Change provider block in `test_provider.go`. For example for Axual Cloud:
```terraform
provider "axual" {
  apiurl   = "https://axual.cloud/api"
  realm    = "<replace with realm name>"
  username = "<replace with username>"
  password = "<replace with password>"
  clientid = "self-service"
  authurl = "https://axual.cloud/auth/realms/dizzl/protocol/openid-connect/token"
  scopes = ["openid", "profile", "email"]
}
```

### How to write tests
- Make sure to test:
    - Creating the resource
    - Updating every field:
        - Optional fields should be able to be removed
        - Sets and Maps
            - test if they can be empty
            - should test if they can contain more than one element
    - Importing
- Deletion is automatic by the acceptance test

### Prevent test automatically deleting resources
- To prevent the test from destroying a resource(for testing):
    - The test will fail but useful for checking in the API what the test actually created

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

### Logging
- Intellij IDEA
    - Edit Configuration for Go Test
    - Add ENV Variable: TF_LOG=INFO
    - Add statements like these into code: tflog.Info(ctx, fmt.Sprintf("delete group successful for group: %q", data.Id.ValueString()))
        - Consider keeping these statements there
- - Does not work if testing with a provider from registry(not locally compiled)

### Debugging
- IntelliJ IDEA
    - Put a breakpoint in resource file, for example resource_group.go.
    - Either put username and password env variables into run configuration(look at Run all `Run acceptance tests.run.xml` in .`run`) or hardcode username and password temporarily.
    - Click on `debug`
- Does not work if testing with a provider from registry(not locally compiled)