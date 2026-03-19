package cli

type exitError struct {
	code    int
	message string
}

func newExitError(code int, message string) error {
	return exitError{
		code:    code,
		message: message,
	}
}

func (e exitError) Error() string {
	return e.message
}

func (e exitError) ExitCode() int {
	return e.code
}
