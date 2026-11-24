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

// releaseName returns the name of the Platform Manager release that brings major.minor. Master builds
// report 16.1.0-SNAPSHOT, but they are released as 17.0.0, so a check for 16.1 names 17.0.0.
func releaseName(major, minor int) string {
	if major == 16 && minor == 1 {
		return "17.0.0"
	}
	return fmt.Sprintf("%d.%d.0", major, minor)
}

// versionUnreadableSummary is the error title when GET /info fails, so the title does not say that the
// Platform Manager is too old.
func versionUnreadableSummary(feature string) string {
	return fmt.Sprintf("Cannot check the Platform Manager version for %s", feature)
}

// requirePlatformVersion adds an error on attribute when the Platform Manager version that
// GetPlatformVersion returned is older than major.minor, or when reading it failed (err). olderDoes
// says what an older Platform Manager does instead.
func requirePlatformVersion(diags *diag.Diagnostics, version string, err error, attribute path.Path, feature string, major, minor int, olderDoes string) {
	summary := fmt.Sprintf("%s needs Platform Manager %s or later", feature, releaseName(major, minor))
	if err != nil {
		diags.AddAttributeError(attribute, versionUnreadableSummary(feature), fmt.Sprintf("%s, and reading the version failed: %s", summary, err))
		return
	}
	if !platformVersionAtLeast(version, major, minor) {
		diags.AddAttributeError(attribute, summary, fmt.Sprintf("This Platform Manager is %s. %s", version, olderDoes))
	}
}
