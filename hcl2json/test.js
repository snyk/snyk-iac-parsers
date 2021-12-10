const { newHCL2JSONParser } = require('./hcl2json');

const hcl2JSONParser = newHCL2JSONParser();

hcl2JSONParser.parse().then(response => {
    console.log(response);
}).catch(err => {
    console.log(err);
});

// Error: native function not implemented: internal/abi.FuncPCABI0