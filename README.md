## Terraform Provider development

Prerequisites
- Install terraform
  - Recommended approach: `brew install terraform`
- Install golang
  - Recommended approach: `brew install go`


Create the file `~/.terraformrc` and add the following to make the provider local installation work:
This points to the locally compiled Terraform Provider on your computer.

Replace `<user>` with the correct username.

```shell
provider_installation {

  dev_overrides {
      "axual.com/hackz/axual" = "/Users/<user>/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

Go to `terraform-provider-axual` directory and run `go mod tidy`. 
This will download the libraries.

Then run `go install` in terraform-provider-axual directory
(or `go build -o $GOPATH/bin/`) to install the provider locally. 
The provider gets installed in `$GOPATH/bin` directory.

Now you can go to `terraform-provider-axual/examples/axual`.

Open the `provider.tf` file and refer to this local binary and provide your configuration.

```terraform
terraform {
  required_providers {
    axual = {
      source  = "axual.com/hackz/axual"
    }
  }
}

# PROVIDER CONFIGURATION
#
# Below example configuration is for when you have deployed Axual Platform locally.

provider "axual" {
  apiurl   = "https://platform.local/api"
  realm    = "axual"
  username = "kubernetes@axual.com"   #- or set using env property export AXUAL_AUTH_USERNAME=
  password = "PLEASE_CHANGE_PASSWORD" #- or set using env property export AXUAL_AUTH_PASSWORD=
  clientid = "self-service"
  authurl  = "https://platform.local/auth/realms/axual/protocol/openid-connect/token"
  scopes   = ["openid", "profile", "email"]
}

# Below example configuration is for when you have deployed Axual Platform in Axual Cloud.

# provider "axual" {
  apiurl   = "https://axual.cloud/api"
  realm    = "PLEASE_CHANGE_REALM"
  username = "PLEASE_CHANGE_USERNAME"
  password = "PLEASE_CHANGE_PASSWORD"
  clientid = "self-service"
  authurl = "https://axual.cloud/auth/realms/PLEASE_CHANGE_REALM/protocol/openid-connect/token"
  scopes = ["openid", "profile", "email"]
# }
```

Now you can run `terraform plan` to test the provider.
> Note you don't need to run `terraform init`.
> 
### Troubleshooting
Usually if Go is installed with brew, you don't need to set any environment variables,
but just in case things are not working, you can try the setting below in `~/.zshrc`.

```shell
export GOPATH=$HOME/go
export GOROOT=“$(brew --prefix golang)/libexec”
export PATH=“$PATH:${GOPATH}/bin:${GOROOT}/bin”
```

## Terraform Documentation

### Generate

- To generate documentation, run this command in terraform-provider-axual directory
```shell
go generate
```
- This command generates documentation based on the templates in the templates' directory.

## Terraform Manifest file(terraform-registry-manifest.json)

- **version** is the numeric version of the manifest format, not the version of our provider.
- **protocol_versions** is the Terraform protocol version. This is set to 6.0 because we are using Terraform Plugin Framework to develop our provider.

## Debugging in IntelliJ IDEA using Delve
- There are four IntelliJ IDEA run configurations in `.run` folder. Open IntelliJ run configuration menu to see them.
- Steps to debug Terraform Provider
  - brew install delve
  - Run "Axual build to _bin" to generate executable in `/go/bin` and name it terraform-provider-axual
  - Run "Axual Delve binary" (Run icon) to start Delve's debugging session on this binary. It looks for terraform-provider-axual executable in `/go/bin`.
  - Run "Axual Go Remote" (Debug icon) to connect IntelliJ to the Delve's debugging session. On the console you will see a string: TF_REATTACH_PROVIDERS='{"axual...
  - Copy this string and use it as environment variable: export TF_REATTACH_PROVIDERS='{"axual...
    - For some reason, it only works in a separate terminal session on MacOS. Also run terraform commands from the same iTerm terminal session.
  - Set a breakpoint and run a command to run the provider like: tf plan or tf apply. It will stop at the breakpoint
  - When done with debugging, remove the env variable: unset TF_REATTACH_PROVIDERS
  - To do the same without generating the binary: Run "Axual Delve project" (Run icon)


## Debugging webclient
- The module axual-webclient-exec is used to test and debug axual-webclient module without any Terraform functionality.

## Logging
- Logging in axual-webclient:
  - log.Println using the module "log"
  - For example:
    - log.Println("strings.NewReader(string(marshal))", strings.NewReader(string(marshal)))
- Logging in internal folder:
  - tflog.Debug(or Info,Trace, etc)
  - Then run TF_LOG=DEBUG(Or INFO,TRACE,ERROR) terraform plan
  - For example:
    - marshal, err := json.Marshal(ApplicationRequest)
    - tflog.Error(ctx, "MARSHAL", map[string]interface{}{
      "MARSHAL": string(marshal),
      })
    - Run: TF_LOG=ERROR terraform apply -auto-approve

## Acceptance tests
### How to run tests
- When using IntellJ IDEA please use and edit this run configuration included in this repo: `.run/Run all the tests.run.xml`
- Need to specify these environment variables. For IntelliJ IDEA click edit configuration -> env variables -> add
  - AXUAL_USERNAME
    - For logging into API
  - AXUAL_PASSWORD
    - For logging into API
  - TF_ACC=1
    - Built in safety env var for accidentally running the tests on a live environment 
  - TF_ACC_TERRAFORM_PATH
    - path to Terraform binary in your local system
  - TF_LOG=INFO
    - Optional but highly recommended
  - Here is a full env variables example: `AXUAL_PASSWORD=<INSERT API PASSWORD>;AXUAL_USERNAME=<INSERT API USERNAME>;TF_ACC=1;TF_ACC_TERRAFORM_PATH=/opt/homebrew/bin/terraform;TF_LOG=INFO`
- Make sure to turn off parallelization for running go tests, because of conflicts when creating shared resources many times
  - Use this go tool argument: `-p 1`
    - For IntelliJ IDEA click edit configuration -> Go tool arguments
- Make sure to turn off test caching, because then we can run the same tests multiple times to test stability without having to change the test.
  - - Use this go tool argument: `-count 1`
- In all test .tf files replace instance UID with a real instance UID. Search globally for `instance = "` and replace the UID in all tests.
- Make sure the certs in the tests match the CA for the Instance, replace them if not.
- Make sure OAUTHBEARER auth method is turned on: in PM conf(`api.available.auth.methods`), Tenant auth method, Instance auth method
  - Needed for testing OAUTHBEARER Application Principal
- Make sure Granular Stream Browse Permissions are turned on for the instance
  - Needed for testing Topic Browse Permissions
- Make sure you replace `data "axual_group" "root_user_group"` with a group name that your logged-in user is a member of in files `axual_topic_browse_permissions_initial.tf` and `axual_topic_browse_permissions_updated.tf`.
- First try to run one acceptance test, before trying to run all the tests. It might happen that if a test fails, you have to manually delete resources using UI.
  - We recommend to try to run in this order:
    - user_resource_test.go
    - topic_data_source_test.go
    - application_deployment_resource_test.go
    - All the tests together
- To run only 1 test in IntelliJ IDEA:
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
```

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
      - should test if they can contain more than 1 element
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