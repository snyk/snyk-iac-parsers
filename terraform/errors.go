package terraform

type CustomError struct {
	message   string
	errors    []error
	userError bool
}

func (err *CustomError) Error() string {
	// TODO: include more details here with user information
	return err.message
}

func isUserError(err error) bool {
	switch e := err.(type) {
	case *CustomError:
		return e.userError
	default:
		return false
	}
}

func createInvalidHCLError(errors []error) *CustomError {
	return &CustomError{
		message:   "Invalid HCL provided",
		errors:    errors,
		userError: true,
	}
}

func createInternalHCLParsingError(errors []error) *CustomError {
	return &CustomError{
		message:   "Unable to convert HCL to JSON object",
		errors:    errors,
		userError: false,
	}
}

func createInternalJSONParsingError(errors []error) *CustomError {
	return &CustomError{
		message:   "Unable to convert JSON object to string",
		errors:    errors,
		userError: false,
	}
}
