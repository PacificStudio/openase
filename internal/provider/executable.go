package provider

// ExecutableResolver resolves executable paths from the local environment.
type ExecutableResolver interface {
	LookPath(name string) (string, error)
}
