package webclient

// ApplicationAccessGrant represents the response from GET /application_access_grants/{id}
// Only fields actually used by the provider are included to reduce maintenance burden
// and avoid breaking changes when the API evolves.
type ApplicationAccessGrant struct {
	Status     string `json:"status"`
	Uid        string `json:"uid"`
	Comment    string `json:"comment"`
	AccessType string `json:"accessType"`
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
	Links struct {
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

// GetApplicationAccessGrantsByAttributeResponse represents the response from the grant search
// GET /application_access_grants/search/findSummariesByAttributes, or findByAttributes before
// Platform Manager 17.0.0.
// Only fields actually used by the provider are included.
type GetApplicationAccessGrantsByAttributeResponse struct {
	Embedded struct {
		ApplicationAccessGrantResponses []ApplicationAccessGrantSummary `json:"applicationAccessGrantResponses"`
	} `json:"_embedded"`
}

// ApplicationAccessGrantRef is the application, stream or environment of a grant in a search result.
type ApplicationAccessGrantRef struct {
	Uid       string `json:"uid"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
}

// ApplicationAccessGrantSummary is one grant of a search result.
type ApplicationAccessGrantSummary struct {
	Uid         string                    `json:"uid"`
	Status      string                    `json:"status"`
	AccessType  string                    `json:"accessType"`
	Application ApplicationAccessGrantRef `json:"application"`
	Stream      ApplicationAccessGrantRef `json:"stream"`
	Environment ApplicationAccessGrantRef `json:"environment"`

	// Embedded holds the application, stream and environment in a findByAttributes answer
	// (before Platform Manager 17.0.0). Remove it with oldGrantSearchPath.
	Embedded *struct {
		Application ApplicationAccessGrantRef `json:"application"`
		Stream      ApplicationAccessGrantRef `json:"stream"`
		Environment ApplicationAccessGrantRef `json:"environment"`
	} `json:"_embedded"`
}

// useEmbedded copies the application, stream and environment of a findByAttributes answer to the plain fields.
func (g *ApplicationAccessGrantSummary) useEmbedded() {
	if g.Embedded != nil {
		g.Application, g.Stream, g.Environment = g.Embedded.Application, g.Embedded.Stream, g.Embedded.Environment
	}
}
