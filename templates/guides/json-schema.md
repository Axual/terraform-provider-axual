---
page_title: "Custom JSON Schema"
---

# Make your IDE know the Axual provider

Your IDE needs a description of the provider (its schema) to give you:

- **Syntax highlighting** of Axual resources and data sources.
- **Validation** of your `resource` and `data` blocks while you type.
- **Auto-completion** and navigation, for example of `data.axual_cluster.alpha.topic_pattern`.

Without it, IntelliJ marks valid code as wrong, for example `Unknown resource: "axual_cluster"` or
`Unresolved reference topic_pattern`. Terraform itself is not affected: `terraform plan` and
`terraform apply` still work.

The IntelliJ Terraform plugin has no Axual schema of its own. It gets the schema in one of two ways:

| Way | Where the schema comes from | Use it when |
|---|---|---|
| 1. Local metadata | IntelliJ runs `terraform providers schema -json` in each folder where `terraform init` was done | Your IntelliJ has the action **Terraform Tools > Generate Providers Metadata**. This is the easy way |
| 2. Global metadata | A JSON file that you put in `~/.terraform.d/metadata-repo/terraform/model/providers/` | You want one schema for every folder, also folders without `terraform init` |

## Way 1: local metadata (IntelliJ)

1. Check that IntelliJ finds Terraform: **Settings > Tools > Terraform Tools**, then
   **Detect and Test** next to the Terraform executable path.
2. Run `terraform init` in your Terraform folder.
3. IntelliJ builds the metadata by itself after `terraform init`. If your files still show
   `Unknown resource`, build it by hand: right-click in a `.tf` file and choose
   **Terraform Tools > Generate Providers Metadata**. The same action is in the menu
   **Code > Terraform Tools**.
4. Wait for the message `Local model has been generated successfully`.

Each folder with its own `terraform init` has its own metadata. Do step 3 again in each folder,
and after every provider upgrade (`terraform init -upgrade`).

## Way 2: global metadata (IntelliJ)

1. Run `terraform init` in a Terraform folder that uses the Axual provider.
2. Write the schema to the folder that IntelliJ reads. For example on macOS and Linux:

   ```shell
   mkdir -p ~/.terraform.d/metadata-repo/terraform/model/providers
   terraform providers schema -json > ~/.terraform.d/metadata-repo/terraform/model/providers/axual.json
   ```

   On Windows, the folder is `%APPDATA%\terraform.d\metadata-repo\terraform\model\providers`.

3. Switch the global metadata on. IntelliJ does not read this folder by default:
   **Settings > Advanced Settings**, section **Terraform**, tick **Use Global metadata**.
4. Restart IntelliJ.

Write the file again after every provider upgrade.

## Check that it works

- The warnings `Unknown resource` and `Unknown data` are gone.
- In a `resource "axual_topic_config"` block, press **Ctrl+Space** on an empty line: the list shows the
  attributes, for example `retention_time`.
- Put the cursor on `axual_topic_config` and press **F1**: the documentation of the resource shows.
