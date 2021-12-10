go get -u github.com/gopherjs/gopherjs
export GOPATH=/Users/teodorasandu/go
export PATH=$PATH:$(go env GOPATH)/bin
export GOPHERJS_GOROOT="$(go env GOROOT)"
gopherjs build .