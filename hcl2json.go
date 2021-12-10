package parsers

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/tmccombs/hcl2json/convert"
	"strings"
)

//func main() {
//	res, _ := HCL2JSON()
//	fmt.Println(res)
//}

func HCL2JSON(file string) (string, error) {
	filename := ""

	// os systemcallls won't work with gopherjs
	//file, err := os.ReadFile(filename)
	//if err != nil {
	//	return "", err
	//}

	// we won't be able to load whole folders because they use os.read
	parsedFile, _ := hclsyntax.ParseConfig([]byte(file), filename, hcl.Pos{Line: 1, Column: 1})

	var options convert.Options = convert.Options{
		Simplify: false,
	}
	// TODO: still using the older version
	hclBytes, err := convert.File(parsedFile, options)
	if err != nil {
		return "", err
	}

	// TODO: stretch item - actually use this in the code (gopherjs)
	return string(hclBytes), nil
}

//	path := "resource.aws_redshift_cluster[denied].logging"
func LineNumber(file string, path string) (string, error) {
	// TODO: they use the filename for the range, in case there are multiple files (but we only have one file)
	filename := ""

	// we won't be able to load whole folders because they use os.read
	parsedFile, _ := hclsyntax.ParseConfig([]byte(file), filename, hcl.Pos{Line: 1, Column: 1})

	// input is gone
	bc, _ := parsedFile.Body.(*hclsyntax.Body)
	// TODO: going through the body a second time after hcl2json
	line := lookup(bc, path)

	return line, nil
}

func lookup(body *hclsyntax.Body, path string) string {
	pathDetails := strings.Split(path, ".")
	// reminder: denied is the name of the resource
	// aws_redshift_cluster[denied].logging -> aws_redshift_cluster.denied.logging
	// TODO: check if Terraform supports more than two labels
	//aws_redshift_cluster[denied][denied]
	//aws_redshift_cluster[denied.denied]

	for _, block := range body.Blocks {
		if block.Type != pathDetails[0] {
			continue
		}

		for i, label := range block.Labels {
			// aws_redshift_cluster
			if i == 0 && label != strings.Split(pathDetails[1], "[")[0] {
				continue
			}

			// allowed/denied
			a := strings.Split(pathDetails[1], "[")[1]
			if i == 1 && label != a[0:len(a)-1] {
				continue
			}

			if i == 1 {
				// TODO: check block and attribute
				for _, block2 := range block.Body.Blocks {
					// aws_redshift_cluster
					if block2.Type != pathDetails[2] {
						continue
					}

					return block2.Body.SrcRange.String()
				}
			}
		}
	}

	return ""
}

// snyk iac test -> resource.aws_redshift_cluster[allowed].encrypted
// {
//     "resource": {
//         "aws_redshift_cluster": {
//             "allowed": {
//                 "encrypted": true,
//                 "logging": {
//                     "enabled": true
//                 }
//             },
//             "denied": {
//                 "encrypted": true,
//                 "logging": {
//                     "enabled": false
//                 }
//             },
//             "denied_2": {
//                 "encrypted": true
//             }
//         }
// }
// Assumption: we only have resource (and no input in front)
// {
// 	"severity": "low",
// 	"resolve": "Set `logging.enabled` attribute to `true`",
// 	"id": "SNYK-CC-TF-136",
// 	"impact": "Audit records may not be available during investigation",
// 	"msg": "resource.aws_redshift_cluster[denied].logging",
// 	"remediation": {
// 	  "cloudformation": "Set `Properties.LoggingProperties` attribute",
// 	  "terraform": "Set `logging.enabled` attribute to `true`"
// 	},
// 	"subType": "Redshift",
// 	"issue": "Amazon Redshift cluster logging is not enabled",
// 	"publicId": "SNYK-CC-TF-136",
// 	"title": "Redshift cluster logging disabled",
// 	"references": [
// 	  "https://docs.aws.amazon.com/redshift/latest/mgmt/db-auditing.html"
// 	],
// 	"isIgnored": false,
// 	"iacDescription": {
// 	  "issue": "Amazon Redshift cluster logging is not enabled",
// 	  "impact": "Audit records may not be available during investigation",
// 	  "resolve": "Set `logging.enabled` attribute to `true`"
// 	},
// 	"lineNumber": 10,
// 	"documentation": "https://snyk.io/security-rules/SNYK-CC-TF-136",
// 	"isGeneratedByCustomRule": false,
// 	"path": [
// 	  "resource",
// 	  "aws_redshift_cluster[denied]",
// 	  "logging"
// 	]
// }
