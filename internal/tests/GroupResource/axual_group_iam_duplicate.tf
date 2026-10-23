resource "axual_group" "team-iam" {
  name          = "testiamgroup9999"
  iam_reference = "tf-test-iam-group-9999-c"
}

resource "axual_group" "team-iam-copy" {
  name          = "testiamgroup9998"
  iam_reference = "tf-test-iam-group-9999-c"
}
