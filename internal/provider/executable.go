package provider

type ExecutableResolver interface {
	LookPath(name string) (string, error)
}
