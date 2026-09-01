package webclient

import (
	"net/http"
	"testing"
)

func TestUidFromLocationHeader(t *testing.T) {
	testCases := []struct {
		desc     string
		location string
		expected string
	}{
		{
			desc:     "Location header of a Flink SQL application deployment",
			location: "https://platform.local/api/application_deployments/e348ea9cd4a84f86b4db847aa3a91d68",
			expected: "e348ea9cd4a84f86b4db847aa3a91d68",
		},
		{
			desc:     "Location header of a KSML application deployment",
			location: "https://platform.local/api/application_deployments/c3741bc7ea3e418bbad2e142bb05b98b",
			expected: "c3741bc7ea3e418bbad2e142bb05b98b",
		},
		{
			desc:     "Location header of a Connector application deployment",
			location: "https://platform.local/api/application_deployments/c94deccbcf7f4841a426196eaa19f487",
			expected: "c94deccbcf7f4841a426196eaa19f487",
		},
		{
			desc:     "trailing slash is ignored",
			location: "https://platform.local/api/application_deployments/c94deccbcf7f4841a426196eaa19f487/",
			expected: "c94deccbcf7f4841a426196eaa19f487",
		},
		{
			desc:     "relative Location header",
			location: "/api/application_deployments/c94deccbcf7f4841a426196eaa19f487",
			expected: "c94deccbcf7f4841a426196eaa19f487",
		},
		{
			desc:     "no Location header",
			location: "",
			expected: "",
		},
	}
	for _, c := range testCases {
		t.Run(c.desc, func(t *testing.T) {
			headers := http.Header{}
			if c.location != "" {
				headers.Set("Location", c.location)
			}
			if uid := uidFromLocationHeader(headers); uid != c.expected {
				t.Fatalf("expected Location %q to yield Uid %q, got %q", c.location, c.expected, uid)
			}
		})
	}
}

func TestUidFromLocationHeaderWithoutHeaders(t *testing.T) {
	if uid := uidFromLocationHeader(nil); uid != "" {
		t.Fatalf("expected no Uid for nil headers, got %q", uid)
	}
}
