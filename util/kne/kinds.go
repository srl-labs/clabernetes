package kne

// VendorModelToClabKind accepts a valid kne Kind and optionally model, and returns the
// corresponding containerlab kind. ex: kne vendor of NOKIA would return srl or "nokia_srlinux";
// kne vendor of CISCO and model "xrd" would return  "cisco_xrd" (if/when that kind exists :p).
func VendorModelToClabKind(kneVendor, kneModel string) string {
	switch kneVendor {
	case "NOKIA":
		return "srl"
	case "ARISTA":
		return "ceos"
	case "CISCO":
		switch kneModel { //nolint:gocritic
		case "xrd":
		}
	case "JUNIPER":
		switch kneModel {
		case "cptx":
		case "ncptx":
		}
	}

	return ""
}
