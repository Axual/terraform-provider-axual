# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is the Terraform Provider for Axual Platform, allowing users to manage Axual resources (applications, topics, schemas, environments, users, groups) through Terraform. The provider is written in Go using the Terraform Plugin Framework.

## Key Commands

### Build and Install
- `go mod tidy` - Download/update Go dependencies
- `go install` - Build and install the provider locally to `$GOPATH/bin`
- `go build -o $GOPATH/bin/` - Alternative build command

### Testing
- `go test -p 1 -count 1 ./internal/tests/...` - Run all acceptance tests (requires TF_ACC=1 environment variable)
- `go test -p 1 -count 1 ./internal/tests/UserResource/user_resource_test.go` - Run a specific test file
- Tests require environment variables: `TF_ACC=1`, `TF_ACC_TERRAFORM_PATH=/opt/homebrew/bin/terraform`, `TF_LOG=INFO`
- Use `-p 1` to disable parallelization (prevents resource conflicts)
- Use `-count 1` to disable test caching

### Documentation
- `go generate` - Generate Terraform provider documentation from templates

### Debugging
- Run with `-debug` flag to enable debugger support
- IntelliJ IDEA run configurations available in `.run/` directory

## Architecture

### Module Structure
- **main.go**: Entry point, initializes the Terraform provider server
- **internal/provider/**: Core provider implementation
  - `provider.go`: Main provider configuration and authentication
  - `resource_*.go`: Resource implementations (CRUD operations)
  - `data_source_*.go`: Data source implementations (read-only)
- **axual-webclient/**: HTTP client library for Axual API interactions
  - Handles authentication, API requests, and response processing
  - Each resource type has corresponding client methods
- **internal/tests/**: Acceptance tests organized by resource type
  - `test_provider.go`: Test provider configuration
  - `test_config.yaml`: Test environment configuration
- **internal/custom-validator/**: Custom validation logic for Terraform attributes

### Provider Configuration
The provider requires authentication configuration pointing to an Axual Platform instance:
- `apiurl`: Base URL for Axual API
- `authurl`: OAuth2 token endpoint
- `realm`, `clientid`, `username`, `password`: Authentication credentials
- `scopes`: OAuth2 scopes

### Resource Pattern
Each Terraform resource follows this pattern:
1. Schema definition with attributes and validators
2. CRUD operations (Create, Read, Update, Delete)
3. Import functionality for existing resources
4. State management using Terraform Plugin Framework types

## Testing Requirements

Before running acceptance tests:
- Update `test_config.yaml` with valid instance names and credentials
- Ensure test user has required roles (Tenant Admin, Application Author, Environment Author, Schema Author, Topic Author)
- Instance must support SSL, SCRAM_SHA_512, and OAUTHBEARER authentication
- Tenant settings must allow resource updates by all group members

## Development Workflow

1. Local provider development uses `~/.terraformrc` with dev_overrides pointing to local binary
2. Make changes to provider code
3. Run `go install` to rebuild
4. Test with `terraform plan/apply` (no `terraform init` needed with dev_overrides)
5. Run acceptance tests before committing
6. Generate documentation with `go generate` if schema changes were made