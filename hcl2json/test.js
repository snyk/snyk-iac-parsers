const { newHCL2JSONParser } = require('./hcl2json');
const { readFileSync } = require('fs')

const file = readFileSync('./test.tf')
console.log(file)
const hcl2JSONParser = newHCL2JSONParser(file, "resource.aws_redshift_cluster[denied].logging");

hcl2JSONParser.parse().then(response => {
    console.log('success')
    console.log(response);
}).catch(err => {
    console.log('failure')
    console.log(err);
}).then(() => {
    return hcl2JSONParser.lineNumber()
}).then(response => {
    console.log('success')
    console.log(response);
}).catch(err => {
    console.log('failure')
    console.log(err);
});

// Error: native function not implemented: internal/abi.FuncPCABI0