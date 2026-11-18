package parser

import "testing"

// Manuals from a firmware repository go under the device type folder name, not the repository name.
func TestDeviceRepoDestinationUsesDeviceName(t *testing.T) {
	repo := DeviceRepoSource{deviceName: "twomes-co2-occupancy-scd41-m5coreink-firmware"}
	got := repo.GetDestinationFilePath("docs/manuals/installation/languages/nl-NL.md")
	want := "devices/twomes-co2-occupancy-scd41-m5coreink-firmware/installation/manufacturer/languages/nl-NL.md"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
