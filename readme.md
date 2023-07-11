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
3. Stream
4. Stream Config
5. Application
6. Application principal
7. Application access grant
### Delete order
All resources can be deleted at once if **depends_on** argument is used like in examples.
Otherwise, this is the correct resource deletion order:
1. Application access grant
2. Application principal
3. Application
4. Stream Config
5. Stream
6. Group
7. User
### Milestone 1
- We currently need to hardcode environment UID because we will develop environment resource in Milestone 2
- Stream
  - Stream key type and value type has to be String/Binary/JSON/Xml. We will support AVRO key and value type in Milestone 2
  - Stream retention_policy has to be string "compact" or "delete'

## Terraform Documentation

### Generate

- To generate documentation run this command in terraform-provider-axual directory
```shell
go generate
```
- This command generates documentation based on the templates in the templates' directory.

## Terraform Manifest

- **version** is the numeric version of the manifest format, not the version of our provider.
- **protocol_versions** is the Terraform protocol version. This is set to 6.0 because we are using Terraform Plugin Framework to develop our provider.