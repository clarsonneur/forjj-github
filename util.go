package main

import (
	"fmt"
	"os"

	"github.com/forj-oss/goforjj"
	"golang.org/x/sys/unix"
)

// Linux support only
func IsWritable(path string) (res bool) {
	return unix.Access(path, unix.W_OK) == nil
}

// check path is writable.
// return false if something is wrong.
func reqCheckPath(name, path string, ret *goforjj.PluginData) bool {

	if path == "" {
		ret.ErrorMessage = name + " is empty."
		return true
	}

	if _, err := os.Stat(path); err != nil {
		ret.ErrorMessage = fmt.Sprintf(name+" mounted '%s' is inexistent.", path)
		return true
	}

	if !IsWritable(path) {
		ret.ErrorMessage = fmt.Sprintf(name+" mounted '%s' is NOT writable", path)
		return true
	}

	return false
}

// verify req data validity.
// return true if something is wrong.
func (g *GitHubStruct) verify_req_fails(ret *goforjj.PluginData, check map[string]bool) bool {
	if v, ok := check["source"]; ok && v {
		if reqCheckPath("source (forjj-source-mount)", g.source_mount, ret) {
			return true
		}
	}

	if v, ok := check["deploy"]; ok && v {
		if reqCheckPath("source (forjj-deploy-mount)", g.deployMount, ret) {
			return true
		}
	}

	if v, ok := check["workspace"]; ok && v {
		if reqCheckPath("workspace (forjj-workspace-mount)", g.workspace_mount, ret) {
			return true
		}
	}

	if v, ok := check["token"]; ok && v {
		if g.token == "" {
			ret.ErrorMessage = fmt.Sprint("github-token is empty - Required")
			return true
		}
	}

	return false // Everything is fine
}

func inStringList(element string, elements ...string) string {
	for _, value := range elements {
		if element == value {
			return value
		}
	}
	return ""
}
