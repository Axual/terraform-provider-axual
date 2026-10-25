resource "axual_group" "team-iam" {
  name          = "testiamgroupds9999"
  iam_reference = "tf-test-iam-group-ds-9999"
}

data "axual_group" "team-iam-imported" {
  name       = axual_group.team-iam.name
  depends_on = [axual_group.team-iam]
}
