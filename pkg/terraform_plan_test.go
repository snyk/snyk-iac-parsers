package parsers

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

const testResultFileExt string = ".resources.json"

func getExpectedResult(planFileName, expectedResultsFolder string) (TerraformScanInput, error) {
	expectedResultFileName := strings.TrimSuffix(planFileName, filepath.Ext(planFileName)) + testResultFileExt
	expectedResultJsonFile, err := ioutil.ReadFile(path.Join(expectedResultsFolder, expectedResultFileName))
	if err != nil {
		return nil, err
	}
	var expectedResult TerraformScanInput
	err = ParseJSON(expectedResultJsonFile, &expectedResult)
	if err != nil {
		return nil, err
	}
	return expectedResult, nil
}

func testPlanFile(t *testing.T, fileName, root string, wg *sync.WaitGroup) {
	defer wg.Done()
	t.Run(fileName, func(t *testing.T) {
		expectedResRoot := path.Join(root, "expected-parser-results/")

		fullScanResultsPath := path.Join(expectedResRoot, "full-scan/")

		deltaScanResultsPath := path.Join(expectedResRoot, "delta-scan/")

		scanModeResultsMap := map[string]string{"full": fullScanResultsPath, "delta": deltaScanResultsPath}

		file, err := ioutil.ReadFile(filepath.Join(root, fileName))
		if err != nil {
			t.Errorf("%v, failed with file %s", err, fileName)
		}
		var planJson TerraformPlanJson
		err = ParseJSON(file, &planJson)
		if err != nil {
			t.Errorf("%v, failed with file %s", err, fileName)
		}
		for scanMode, expectedResultsFolder := range scanModeResultsMap {
			t.Run(fmt.Sprintf("%v scan for %s", scanMode, fileName), func(t *testing.T) {
				parsedPlan := parseTerraformPlan(planJson, "full" == scanMode)
				expectedResult, err := getExpectedResult(fileName, expectedResultsFolder)
				if err != nil {
					t.Errorf("%v, failed with file %s, with scan type %s", err, fileName, scanMode)
				}
				if !reflect.DeepEqual(parsedPlan, expectedResult) {
					t.Logf("file %s didn't pass %s scan!", fileName, scanMode)
					t.FailNow()
				}
			})
		}
	})
}

func TestTerraformPlanParser(t *testing.T) {
	root := "../test/fixtures/terraform-plans/"
	files, err := ioutil.ReadDir(root)
	if err != nil {
		t.Error(err)
	}
	wg := sync.WaitGroup{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		wg.Add(1)
		go testPlanFile(t, file.Name(), root, &wg)
	}
	wg.Wait()
}
