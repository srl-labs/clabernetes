package kne

// VendorModelToImage accepts a valid kne Kind and optionally model, and returns the
// corresponding image to use in containerlab. Ideally users will specify images in their topology
// file though as some images are not publicly available (unlike srl!).
func VendorModelToImage(kneVendor, kneModel string) string {
	switch kneVendor {
	case "NOKIA":
		return "ghcr.io/nokia/srlinux"
	case "ARISTA":
	case "CISCO":
		switch kneModel { //nolint:revive
		}
	case "JUNIPER":
		switch kneModel { //nolint:revive
		}
	}

	return ""
}
