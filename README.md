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
  username = "kubernetes@axual.com"   #- or set using env property export AXUAL_USERNAME=
  password = "PLEASE_CHANGE_PASSWORD" #- or set using env property export AXUAL_PASSWORD=
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

- Please read how to run tests from `DEVELOPERS.md`