package app

// parseBuildSuperuserSetMutationStatus resolves one typed mutation status from create-vs-update state.
func parseBuildSuperuserSetMutationStatus(isParseExisting bool) string {
	if isParseExisting {
		return "updated"
	}
	return "created"
}
