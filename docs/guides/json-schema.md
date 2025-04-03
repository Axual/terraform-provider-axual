---
page_title: "Custom JSON Schema"
---

- This guide shows you how to generate a custom JSON schema from your Terraform provider to integrate it with your IDE.
- Using a custom JSON schema enhances your Terraform usage experience by providing:
  - **Syntax Highlighting**
  - The IDE **validates** your resources and data blocks as you write them.
  - Improves navigation and auto-completion by **recognizing custom provider references**.

### Generating the JSON Schema

- In Terraform configuration directory run:

```shell
terraform init
```

- Generate the provider schema and redirect the JSON schema to the appropriate directory. For example in MacOS:
```shell
mkdir -p ~/.terraform.d/metadata-repo/terraform/model/providers
terraform providers schema -json | jq -r '.' > ~/.terraform.d/metadata-repo/terraform/model/providers/axual.json
```

- After saving the schema, restart your IDE to apply the new schema.
- Whenever you upgrade the Terraform provider version, regenerate the schema by repeating the steps above.
