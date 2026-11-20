package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// platformVersionAtLeast reports whether a Platform Manager version, as GET /info returns it (for
// example "16.0.3" or "16.1.0-SNAPSHOT"), is major.minor or later. A version it cannot read counts as
// older.
func platformVersionAtLeast(version string, major, minor int) bool {
	var gotMajor, gotMinor int
	if _, err := fmt.Sscanf(version, "%d.%d", &gotMajor, &gotMinor); err != nil {
		return false
	}
	return gotMajor > major || gotMajor == major && gotMinor >= minor
}

// requirePlatformVersion adds an error on attribute when the Platform Manager version that
// GetPlatformVersion returned is older than major.minor, or when reading it failed (err). olderDoes
// says what an older Platform Manager does instead.
func requirePlatformVersion(diags *diag.Diagnostics, version string, err error, attribute path.Path, feature string, major, minor int, olderDoes string) {
	summary := fmt.Sprintf("%s needs Platform Manager %d.%d.0 or later", feature, major, minor)
	if err != nil {
		diags.AddAttributeError(attribute, summary, fmt.Sprintf("Reading the Platform Manager version failed: %s", err))
		return
	}
	if !platformVersionAtLeast(version, major, minor) {
		diags.AddAttributeError(attribute, summary, fmt.Sprintf("This Platform Manager is %s. %s", version, olderDoes))
	}
}
