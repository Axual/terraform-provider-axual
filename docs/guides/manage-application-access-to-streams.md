---
page_title: "Managing application access to streams"
---

Managing access through Terraform is not straightforward.
Important concepts to note first,
- application_access_grant: resource to make request for application to access(consume/produce) a stream in an environment.
- application_access_grant_approval: resource to approve and revoke application_access_grant
- application_access_grant_rejection: resource to reject application_access_grant


#### Application Owner
An Application owner can request access to a Stream by creating a `application_access_grant` resource.
If the `application_access_grant` is auto approved in the specified environment, then no further action is required on the Application Owner's part. Access is fully granted.
If the `application_access_grant` requires approval from Stream Owner, the section on Stream Owner shows them how to do it.
An `application_access_grant` can be cancelled by deleting the resource. This is only possible if it is not pending approval.

#### Stream Owner
A Stream owner can reject access to a Stream by creating a `application_access_grant_rejection` resource. An optional reason can be provided.
A Stream owner can approve access to a Stream by creating a `application_access_grant_approval` resource. 
A Stream owner can revoke access to a Stream by deleting the corresponding `application_access_grant_approval` resource.
If the stream was auto approved, a `application_access_grant_approval` resource has to be created for the `application_access_grant`, and then destroyed in order to revoke the grant.
