package main

// Run in docker context: GOPATH is /go
//go:generate go build -o /go/bin/forjj-genapp forjj-github/vendor/github.com/forj-oss/goforjj/genapp
//go:generate /go/bin/forjj-genapp github.yaml vendor/github.com/forj-oss/goforjj/genapp
