package webclient

// ApplicationAccessGrant represents the response from GET /application_access_grants/{id}
// Only fields actually used by the provider are included to reduce maintenance burden
// and avoid breaking changes when the API evolves.
type ApplicationAccessGrant struct {
	Status  string `json:"status"`
	Uid     string `json:"uid"`
	Comment string `json:"comment"`
	Links   struct {
		Revoke struct {
			Href string `json:"href"`
		} `json:"revoke"`
		Approve struct {
			Href string `json:"href"`
		} `json:"approve"`
		Cancel struct {
			Href string `json:"href"`
		} `json:"cancel"`
		Deny struct {
			Href string `json:"href"`
		} `json:"deny"`
	} `json:"_links"`
}

// ApplicationAccessGrantRevoke is used for revoke/deny API requests
type ApplicationAccessGrantRevoke struct {
	Reason      string `json:"reason"`
	Environment string `json:"environment"`
}

// ApplicationAccessGrantRequest is used for creating a new grant
type ApplicationAccessGrantRequest struct {
	ApplicationId string `json:"applicationId"`
	StreamId      string `json:"streamId"`
	EnvironmentId string `json:"environmentId"`
	AccessType    string `json:"accessType"`
}

// ApplicationAccessGrantResponse represents the response from POST /application_access_grants
// Only fields actually used by the provider are included.
type ApplicationAccessGrantResponse struct {
	Uid         string `json:"uid"`
	Status      string `json:"status"`
	Environment struct {
		Id string `json:"id"`
	} `json:"environment"`
}

// GetApplicationAccessGrantsByAttributeResponse represents the response from the search endpoint
// GET /application_access_grants/search/findByAttributes
// Only fields actually used by the provider are included.
type GetApplicationAccessGrantsByAttributeResponse struct {
	Embedded struct {
		ApplicationAccessGrantResponses []struct {
			AccessType string `json:"accessType"`
			Uid        string `json:"uid"`
			Status     string `json:"status"`
			Embedded   struct {
				Environment struct {
					Uid string `json:"uid"`
				} `json:"environment"`
				Application struct {
					Uid string `json:"uid"`
				} `json:"application"`
				Stream struct {
					Uid string `json:"uid"`
				} `json:"stream"`
			} `json:"_embedded"`
		} `json:"applicationAccessGrantResponses"`
	} `json:"_embedded"`
	Page struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}
