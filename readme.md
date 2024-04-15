## Terraform Provider development

Create the file `~/.terraformrc` and add the following to make the provider local installation work, put the correct path, for the record `~/go/bin` did not seem to work... Replace `daniel` with the correct username.
```shell
provider_installation {

  dev_overrides {
      "axual.com/hackz/axual" = "/Users/daniel/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

Refer to this local binary from 00_provider.tf
```shell
terraform {
  required_providers {
    axual = {
      source  = "axual.com/hackz/axual"
    }
  }
}
```

Make sure `GO111MODULE` is set to `on` by running the command `go env GO111MODULE`. If it is not on, change it with the command `go env -w GO111MODULE="on"`.

Go to `terraform-provider-axual` directory and run `go mod tidy`. This will download the libraries.

Then run `go install` in terraform-provider-axual directory (or `go build -o $GOPATH/bin/`) to install the provider locally. The provider gets installed in `$GOPATH/bin` directory.

Now you can go to `terraform-provider-axual/examples/axual` and run `terraform plan` to test the provider. Note you don't need to run `terraform init`.

Usually if Go is installed with brew, you don't need to set any environment variables but just in case things are not working, you can try setting below in `~/.zshrc`.

```shell
export GOPATH=$HOME/go
export GOROOT=“$(brew --prefix golang)/libexec”
export PATH=“$PATH:${GOPATH}/bin:${GOROOT}/bin”
```

## Terraform configuration
### Create order
All resources can be created at once if **depends_on** argument is used like in examples.
Otherwise, this is the correct resource creation order:
1. User
2. Group
3. Topic
4. Topic Config
5. Application
6. Application principal
7. Application access grant
### Delete order
All resources can be deleted at once if **depends_on** argument is used like in examples.
Otherwise, this is the correct resource deletion order:
1. Application access grant
2. Application principal
3. Application
4. Topic Config
5. Topic
6. Group
7. User
### Milestone 1 Features
- Added support for Tenant, User, Group
- Added support for Application, ApplicationPrincipal
- Added support for Topic, TopicConfig
- Topic key type and value type has to be String/Binary/JSON/XML
  Topic retention_policy has to be string “compact” or “delete’

### Milestone 2 Features
- Added support for Environment
- Added support for Topic Access

### Work in progress
- Support schema and schemaVersion resource
- Support AVRO topic and AVRO topicConfig
- Data Sources

## Terraform Documentation

### Upgrading to version 2.0.0+
In version 2.0.0 all wording of stream has been changed to topic. i.e the axual resource `axual_stream` has been renamed to `axual_topic`. To update from versions 1.x.x to 2.0.0 or above you need to 
1. Replace all references of stream in state files with topic
 below are commands to run for the provided examples project, you need to run the same commands in the root of your project
 ```shell
  sed -i='' "s/axual_stream_config/axual_topic_config/" terraform.tfstate
  sed -i='' "s/axual_stream/axual_topic/" terraform.tfstate
  sed -i='' "s/\"stream\":/\"topic\":/" terraform.tfstate
 ```
2. Replace all references of stream in configuration files with topic
 ```shell
sed -i='' "s/axual_stream_config/axual_topic_config/" main.tf
sed -i='' "s/axual_stream/axual_topic/" main.tf
sed -i='' "s/stream =/topic =/" main.tf
 ```

### Generate

- To generate documentation run this command in terraform-provider-axual directory
```shell
go generate
```
- This command generates documentation based on the templates in the templates' directory.

## Terraform Manifest

- **version** is the numeric version of the manifest format, not the version of our provider.
- **protocol_versions** is the Terraform protocol version. This is set to 6.0 because we are using Terraform Plugin Framework to develop our provider.

## Debugging in IntelliJ IDEA using Delve
- There are 4 IntelliJ IDEA run configurations in .run folder. Just open IntelliJ run configuration menu to see them.
- Steps to debug Terraform Provider
  - brew install delve
  - Run "Axual build to _bin" to generate executable in /go/bin and name it terraform-provider-axual
  - Run "Axual Delve binary"(Run icon) to start Delve's debugging session on this binary. It looks for terraform-provider-axual executable in /go/bin.
  - Run "Axual Go Remote"(Debug icon) to connect IntelliJ to the Delve's debugging session. On the console you will see a string: TF_REATTACH_PROVIDERS='{"axual...
  - Copy this string and use it as environment variable: export TF_REATTACH_PROVIDERS='{"axual...
    - For some reason, it only works in a separate terminal session on MacOS. Also run terraform commands from the same iTerm terminal session.
  - Set a breakpoint and run a command to run the provider like: tf plan or tf apply. It will stop at the breakpoint
  - When done with debugging, remove the env variable: unset TF_REATTACH_PROVIDERS
  - To do the same without generating the binary: Run "Axual Delve project"(Run icon)


## Debugging webclient
- The module axual-webclient-exec is used to test and debug axual-webclient module without any Terraform functionality.

## Logging
- Logging in axual-webclient:
  - log.Println using the module "log"
  - For example:
    - log.Println("strings.NewReader(string(marshal))", strings.NewReader(string(marshal)))
- Logging in internal folder:
  - tflog.Debug(or Info,Trace etc)
  - Then run TF_LOG=DEBUG(Or INFO,TRACE,ERROR) terraform plan
  - For example:
    - marshal, err := json.Marshal(ApplicationRequest)
    - tflog.Error(ctx, "MARSHAL", map[string]interface{}{
      "MARSHAL": string(marshal),
      })
    - Run: TF_LOG=ERROR terraform apply -auto-approve
