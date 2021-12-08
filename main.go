package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/tmccombs/hcl2json/convert"
)

func main() {
	filename := "test.tf"

	file, err := os.ReadFile(filename)
	if err != nil {
		fmt.Errorf(err.Error())
	}
	parsedFile, _ := hclsyntax.ParseConfig(file, filename, hcl.Pos{Line: 1, Column: 1})

	var options convert.Options = convert.Options{
		Simplify: false,
	}
	// TODO: still using the older version
	hclBytes, err := convert.File(parsedFile, options)
	if err != nil {
		fmt.Errorf("convert to HCL: %w", err)
	}

	// TODO: stretch item - actually use this in the code (gopherjs)
	fmt.Println(string(hclBytes))

	bc, _ := parsedFile.Body.(*hclsyntax.Body)
	//fmt.Println(bc)
	for _, block := range bc.Blocks {
		//fmt.Println(block)
		for _, att := range block.Body.Attributes {
			fmt.Printf("%s, range: %s \n", att.Name, att.SrcRange)
		}
	}

	// TODO: lookup function that looks for the fields in the msg/path and iterated through the body
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
