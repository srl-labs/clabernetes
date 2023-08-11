package kne

// VendorModelToClabKindMapper accepts a valid kne Kind (as in node kind, ex: kne vendor of NOKIA
// would return srl or "nokia_srlinux"; kne vendor of CISCO and model "xrd" would return
// "cisco_xrd" (if/when that kind exists :p).
func VendorModelToClabKindMapper(kneVendor, kneModel string) (string, error) {
	// TODO obviously.
	_, _ = kneVendor, kneModel

	return "srl", nil
}
