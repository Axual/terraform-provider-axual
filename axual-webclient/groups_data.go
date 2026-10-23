package webclient

import "encoding/json"

type GroupResponse struct {
	Name         string `json:"name"`
	EmailAddress struct {
		Email string `json:"email"`
	} `json:"emailAddress"`
	PhoneNumber interface{} `json:"phoneNumber"`
	Uid         string      `json:"uid"`
	// IamReference is the group in the IAM system this group is linked to. Empty for an
	// Axual-managed group: Platform Manager leaves the field out.
	IamReference string `json:"iamReference"`
	Embedded     struct {
		Managers []struct {
			Uid string `json:"uid"`
		} `json:"managers"`
		Members []struct {
			Uid string `json:"uid"`
		} `json:"members"`
	} `json:"_embedded"`
}

type GroupRequest struct {
	Name         string      `json:"name,omitempty"`
	EmailAddress interface{} `json:"emailAddress"`
	PhoneNumber  interface{} `json:"phoneNumber"`
	Members      []string    `json:"members"`
	Managers     []string    `json:"managers"`
	IamReference string      `json:"iamReference,omitempty"`

	// SkipMembers leaves members and managers out of the body. For an IAM-managed group the IAM
	// system decides who is a member, so the provider must not send (or clear) the stored lists.
	SkipMembers bool `json:"-"`
}

func (g GroupRequest) MarshalJSON() ([]byte, error) {
	type plain GroupRequest
	if !g.SkipMembers {
		return json.Marshal(plain(g))
	}
	return json.Marshal(struct {
		plain
		Members  *[]string `json:"members,omitempty"`
		Managers *[]string `json:"managers,omitempty"`
	}{plain: plain(g)})
}

// ExternalizeGroupRequest links an Axual-managed group to a group in the IAM system.
type ExternalizeGroupRequest struct {
	IamReference string `json:"iamReference"`
}

type GetGroupByNameResponse struct {
	Embedded struct {
		Groups []struct {
			Name string `json:"name"`
			Uid  string `json:"uid"`
		} `json:"groups"`
	} `json:"_embedded"`
}
