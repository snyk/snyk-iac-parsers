package parsers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"

	"github.com/pkg/errors"
)

type ResourceActions []string

type TerraformPlanResource struct {
	Address string      // "aws_cloudwatch_log_group.terra_ci",
	Mode    string      // "managed",
	Type    string      // "aws_cloudwatch_log_group",
	Name    string      // "terra_ci",
	Index   interface{} // Can be either an integer or a string (e.g. 1, "10.0.101.0/24", "rtb-00cf8381520103cfb")
}

type TerraformPlanResourceChange struct {
	TerraformPlanResource
	Change struct {
		Actions ResourceActions
		Before  map[string]interface{} // will be null when the action is `create`
		After   map[string]interface{} // will be null when then action is `delete`
	}
	Expressions interface{} `json:"expressions"`
}

type TerraformPlanModule struct {
	Resources []TerraformPlanResourceChange `json:"resources"`
}

type TerraformPlanConfiguration struct {
	RootModule TerraformPlanModule `json:"root_module"`
}

type TerraformPlanJson struct {
	ResourceChanges []TerraformPlanResourceChange `json:"resource_changes"`
	Configuration   TerraformPlanConfiguration    `json:"configuration"`
}

type TerraformScanInput map[string]map[string]map[string]interface{}

func getValidResourceActionsForDeltaScan() []ResourceActions {
	return []ResourceActions{{`create`}, {`update`}, {`create`, `delete`}, {`delete`, `create`}}
}

func getValidResourceActionsForFullScan() []ResourceActions {
	return append(getValidResourceActionsForDeltaScan(), ResourceActions{`no-op`})
}

func parseTerraformPlan(planJson TerraformPlanJson, isFullScan bool) TerraformScanInput {
	scanInput := TerraformScanInput{
		"resource": map[string]map[string]interface{}{},
		"data":     map[string]map[string]interface{}{},
	}
	for _, resource := range planJson.ResourceChanges {
		// checks if valid action, if invalid skip loop iteration
		if !isValidResourceActions(resource.Change.Actions, isFullScan) {
			continue
		}
		// get correct mode for scanInput
		var mode string
		if resource.Mode == "data" {
			mode = "data"
		} else {
			mode = "resource"
		}
		// even though we only support resource or data options, we do this as a sanity check
		if _, ok := scanInput[mode]; ok {
			// scanInput's mode is set, add item to mode
			if _, ok := scanInput[mode][resource.Type]; ok {
				// scanInput[mode][resource.Type] resource type already created, adding another resource under it with a new name
				scanInput[mode][resource.Type][getResourceName(resource)] = resource.Change.After
			} else {
				// set new resource type with its values
				scanInput[mode][resource.Type] = map[string]interface{}{getResourceName(resource): resource.Change.After}
			}
		}
	}

	// check root module for references in first depth of attributes
	for _, resource := range planJson.Configuration.RootModule.Resources {
		// don't care about references in data sources for time being
		if resource.Mode == "data" {
			continue
		}
		mode := "resource"
		// only update the references in resources that have some resolved attributes already
		if resolvedResource, ok := scanInput[mode][resource.Type][getResourceName(resource)].(map[string]interface{}); ok && resolvedResource != nil {
			expressions := getExpressions(resource.Expressions)
			for k, v := range expressions {
				// only add non existing attributes. If we already have resolved value do not overwrite it with reference
				if _, ok := resolvedResource[k]; !ok {
					resolvedResource[k] = v
				}
			}
			scanInput[mode][resource.Type][getResourceName(resource)] = resolvedResource
		}
	}

	return scanInput
}

func getExpressions(expressions interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	// expressions can be nested. we are only doing 1 depth to resolve top level depenencies
	expressionMap, ok := expressions.(map[string]interface{})
	if !ok {
		return nil
	}
	for k, v := range expressionMap {
		referenceKey, ok := getReference(v)
		if ok {
			result[k] = referenceKey
		}
	}
	return result
}

// this is very naive implementation
// the referenences can be composed of number of keys
// we only going to use the first reference for time being
func getReference(value interface{}) (interface{}, bool) {
	v, ok := value.(map[string]interface{})
	if !ok {
		return "", false
	}
	// we are only interested with "references" values
	referencesInt, ok := v["references"]
	if !ok {
		return "", false
	}
	references, ok := referencesInt.([]interface{})
	if !ok {
		return "", false
	}
	return references[0], true
}

func getResourceName(resource TerraformPlanResourceChange) string {
	if resource.Index == nil {
		return resource.Name
	} else {
		// if an index field is present, use name + index to diffrentiate multi-instance resources
		// e.g resource 1 with same type & name but different index
		// "type": "aws_route",
		// "name": "private",
		// "index": "rtb-00cf8381520103cfb",
		// e.g resource 2 with same type & name but different index
		// "type": "aws_route",
		// "name": "private",
		// "index": "rtb-030b64d80cb5e9da7",
		var indexKey string = mapResourceIndexToStringKey(resource.Index)
		return fmt.Sprintf(`%s["%s"]`, resource.Name, indexKey)
	}
}

func mapResourceIndexToStringKey(resourceIndex interface{}) string {
	var indexType reflect.Kind = reflect.TypeOf(resourceIndex).Kind()
	var indexKey string
	if indexType == reflect.Int {
		indexKey = strconv.Itoa(resourceIndex.(int))
	} else if indexType == reflect.String {
		indexKey = resourceIndex.(string)
		// In some cases the JSON Unmarshal will decode an Integer as a Float, therefore the two following checks
	} else if indexType == reflect.Float32 {
		indexKey = strconv.Itoa(int(resourceIndex.(float32)))
	} else if indexType == reflect.Float64 {
		indexKey = strconv.Itoa(int(resourceIndex.(float64)))
	} else {
		// If some unknown value was used here, we'll generate some random integer.
		indexKey = strconv.Itoa(rand.Intn(10000))
	}

	return indexKey
}

func isValidResourceActions(resourceAction ResourceActions, isFullScan bool) bool {
	var validActions []ResourceActions
	if isFullScan {
		validActions = getValidResourceActionsForFullScan()
	} else {
		validActions = getValidResourceActionsForDeltaScan()
	}
	for _, validAction := range validActions {
		if reflect.DeepEqual(validAction, resourceAction) {
			return true
		}
	}
	return false
}

func ParseTerraformPlan(p []byte, v *interface{}) error {
	var tfPlanJson TerraformPlanJson
	if err := json.Unmarshal(p, &tfPlanJson); err != nil {
		return errors.Wrap(err, "failed to parse terraform-plan json payload")
	}
	// Currently being used only by Terraform Cloud integration
	// It was decided that using Full Scan as the default scan is the right approach
	// In the future this will be configurable
	*v = parseTerraformPlan(tfPlanJson, true)
	return nil
}
