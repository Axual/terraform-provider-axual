# Group members and managers are sent as bare uids

Platform Manager's group API names each member and manager with a typed URI —
`/users/{uid}` for a person, `/service-accounts/{uid}` for a service account —
and rejects the write when the collection disagrees with the row. The provider
cannot make that claim: `axual_group.members` and `axual_group.managers` hold
bare uids, in configuration and in state alike, and on create there is no prior
read to echo back. So the provider sends the **bare uid** and lets Platform
Manager resolve which kind it is.

## Considered options

| | Approach | Why not |
|---|---|---|
| a | Keep each member's type in Terraform state and use it on writes | Create has no prior state to read it from |
| b | Ask the API what each uid is before writing (`GET /service-accounts/{uid}`) | One sequential round trip per member *and* per manager — 2N calls per group write, on create and update, nearly all of them 404s. A flaky probe fails an apply that previously made no network call at all |
| c | Take typed references in HCL (`axual_service_account.x.id`) | Needs an `axual_service_account` resource, which does not exist, and changes the schema |
| d | Send `/users/{uid}` and retry as `/service-accounts/{uid}` on rejection | Platform Manager returns the same error for a wrong collection and an unknown uid, so a typo triggers a pointless retry and then reports the wrong cause |
| **e** | **Send a bare uid** | **Chosen** |

(b) was built and reviewed before (e) was found. The review objected to its cost,
and fixing it properly meant doubling it, because managers resolve through the
same code path as members.

## Why this was available

A bare uid was already accepted, and already meant "a person". Asking Platform
Manager to resolve it instead was one method — `GroupMembershipService.expectedTypeFor`
returns no expected type when the reference carries no collection (AXPD-12189).

The meaning change was safe only because of timing. Service accounts and the
typed-URI check shipped in the same, still-unreleased Platform Manager commit,
and before it every user row was a person — so no released version ever
interpreted a bare uid in a way this contradicts. Landing it one release later
would have left a version where a bare uid means "person", and the provider
would have had to carry (b) for it anyway.

## Consequences

- A typed URI still works and is still validated. A client that read the group
  first, and so holds the typed `self` href, may echo it back.
- The provider requires Platform Manager 15.1.0 or later to put a service
  account in a group. Groups of people work on every version, before and after.
- Nothing in the provider needs to know what a uid refers to, so no member type
  is stored in state and no lookup exists to keep correct.
