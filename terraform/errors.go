package terraform

import "fmt"

type CustomError struct {
	message   string
	errors    []error
	userError bool
}

func (err *CustomError) Error() string {
	return err.message
}

func GenerateDebugLogs(err error) string {
	customError, ok := err.(*CustomError)

	debugging := ""
	if ok {
		for _, e := range customError.errors {
			debugging = fmt.Sprintf("%s\n%s", debugging, e.Error())
		}
	}
	return debugging
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

func createDisallowedFunctionError(errors []error) *CustomError {
	return &CustomError{
		message:   "Attempted to call disallowed function",
		errors:    errors,
		userError: true,
	}
}
