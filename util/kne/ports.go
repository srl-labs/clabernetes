package kne

// VendorModelToDefaultPorts accepts a valid kne Kind and optionally model and returns the
// default ports configured in kne. Since we are not actually "doing" kne things, and this is no
// longer defined in the topos we just have this mapping we need to keep updated. The value returned
// is a slice of strings that mirror containerlab port definitions.
func VendorModelToDefaultPorts(kneVendor, kneModel string) []string {
	_ = kneModel

	switch kneVendor { //nolint:gocritic
	// maybe some others matter at some point? looks like mostly just standard stuff now tho?
	default:
		return []string{
			"21022:22",
			"21443:443",
			"29559:9559",  // p4rt
			"57400:57400", // gnmi/gnoi
			"57401:57401", // gribi
		}
	}
}
