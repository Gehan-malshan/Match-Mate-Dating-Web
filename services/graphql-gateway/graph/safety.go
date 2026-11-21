package graph

// The Account service issues this exact restricted role for suspended members.
func onlyAppeal(roles []string) bool {
	return len(roles) == 1 && roles[0] == "appeal"
}
