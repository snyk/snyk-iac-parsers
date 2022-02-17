package terraform

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateDebugLogsNotCustomError(t *testing.T) {
	actual := GenerateDebugLogs(errors.New("Test"))
	assert.Equal(t, "", actual)
}

func TestGenerateDebugLogsCustomErrorWithNoErrors(t *testing.T) {
	actual := GenerateDebugLogs(&CustomError{
		message: "Test",
		errors:  []error{},
	})
	assert.Equal(t, "", actual)
}

func TestGenerateDebugLogsCustomErrorWithErrors(t *testing.T) {
	actual := GenerateDebugLogs(&CustomError{
		message: "Test",
		errors: []error{
			errors.New("Error1"),
			errors.New("Error2"),
		},
	})
	assert.Equal(t, "\nError1\nError2", actual)
}

func TestIsUserError(t *testing.T) {
	assert.False(t, isUserError(errors.New("Test")))
	assert.False(t, isUserError(&CustomError{
		message: "Test",
		errors:  []error{},
	}))
}
