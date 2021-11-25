package parsers

import (
	"context"
	"encoding/json"
	// "runtime"

	// "github.com/snyk/cloud-config-policy-engine/core/instrumentation"

	"github.com/pkg/errors"
	hcl2jsonConverter "github.com/tmccombs/hcl2json/convert"
)

// ParseHCL2 unmarshals HCL files that are written using
// version 2 of the HCL language and return parsed file content.
func ParseHCL2(ctx context.Context, p []byte, v interface{}) (err error) {
	// TODO: Look into using plain github.com/hashicorp/hcl/v2
	// instead to avoid the JSON intermediary format.
	// TODO Remove the following code which is used for capturing more details about errors happening while parsing hcl2
	//defer func() {
	//	// we encountered an issue where the underlying parser library panics when a parser error happens for a bad hcl file
	//	// The go-cty library panics here https://github.com/zclconf/go-cty/blob/v1.6.1/cty/value_ops.go#L1234
	//	// when trying to convert a null value as a string. We chose to recover and not panic for that.
	//	if r := recover(); r != nil {
	//		stackBuf := make([]byte, 8*1024)
	//		stackBytesWritten := runtime.Stack(stackBuf, false)
	//		stackBuf = stackBuf[0:stackBytesWritten]
	//
	//		event := instrumentation.FromContext(ctx)
	//		event.Store("panic_err", r)
	//		event.Store("panic_stack", instrumentation.UnredactedString(stackBuf))
	//
	//		err = errors.New("hcl2 to json conversion failed")
	//	}
	//}()
	jsonBytes, err := hcl2jsonConverter.Bytes(p, "", hcl2jsonConverter.Options{})
	if err != nil {
		return errors.Wrap(err, "hcl2 to json conversion failed")
	}

	if err := json.Unmarshal(jsonBytes, v); err != nil {
		return errors.Wrap(err, "unmarshal hcl2 json failed")
	}

	return nil
}
