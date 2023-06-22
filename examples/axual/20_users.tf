resource "axual_user" "john" {
  first_name    = "John"
  last_name     = "Doe"
  email_address = "john.doe@example.com"
  phone_number = "+37253412551"
  roles         = [
    { name = "TENANT_ADMIN" },
  ]
}

resource "axual_user" "jane" {
  first_name    = "Jane"
  last_name     = "Walker"
  email_address = "jane.walker@example.com"
  phone_number = "+37253412553"
  roles         = [
    { name = "TENANT_ADMIN" },
  ]
}


resource "axual_user" "green" {
  first_name    = "Green"
  last_name     = "Stones"
  email_address = "green.stones@example.com"
  phone_number = "+37253412552"
  roles         = [
    { name = "TENANT_ADMIN" },
  ]
}


output "jane_id" {
  description = "Jane's ID"
  value = axual_user.jane.id
}

output "jane_last_name" {
  description = "Jane's Last Name"
  value = axual_user.jane.last_name
}