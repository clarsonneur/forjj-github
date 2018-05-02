// This file has been created by "go generate" as initial code. go generate will never update it, EXCEPT if you remove it.

// So, update it for your need.
package main

// You can remove following comments.
// It has been designed fo you, to implement the core of your plugin task.
//
// You can use use it to write your own plugin handler for additional functionnality
// Like Index which currently return a basic code.

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/forj-oss/goforjj"
)

const github_file = "github.yaml"

// Do creating plugin task
// req_data contains the request data posted by forjj. Structure generated from 'github.yaml'.
// ret_data contains the response structure to return back to forjj.
//
// By default, if httpCode is not set (ie equal to 0), the function caller will set it to 422 in case of errors (error_message != "") or 200
func DoCreate(w http.ResponseWriter, r *http.Request, req *CreateReq, ret *goforjj.PluginData) (httpCode int) {
	instance := req.Forj.ForjjInstanceName
	gws := GitHubStruct{
		source_mount: req.Forj.ForjjSourceMount,
		token:        req.Objects.App[instance].Token,
	}
	check := make(map[string]bool)
	check["token"] = true
	check["source"] = true
	log.Printf("Checking parameters : %#v", gws)

	//ensure source path is writeable
	if gws.verify_req_fails(ret, check) {
		return
	}

	if a, found := req.Objects.App[instance]; !found {
		ret.Errorf("Internal issue. Forjj has not given the Application information for '%s'. Aborted.")
		return
	} else {
		gws.app = &a
	}

	log.Print("Checking github connection.")

	if gws.github_connect(gws.app.Server, ret) == nil {
		return
	}

	// Build gws.github_source yaml structure.
	if err := gws.create_yaml_data(req, ret); err != nil {
		ret.Errorf("Unable to create. %s", err)
		return
	}

	// A create won't be possible if repo requested already exist. The Update is the only possible option.
	// The list of repository found are listed and returned in the answer.
	if err := gws.repos_exists(ret); err != nil {
		ret.Errorf("%s\nUnable to 'create' your forge when github already has an infra repository created. Clone it and use 'update' instead.", err)
		return 419
	}

	// A create won't be possible if source files already exist. The Update is the only possible option.
	log.Print("Checking Infrastructure code existence.")
	sourceRepo := req.Forj.ForjjSourceMount
	sourcePath := path.Join(sourceRepo, req.Forj.ForjjInstanceName)
	sourceFile := path.Join(sourcePath, github_file)

	deployRepo := path.Join(req.Forj.ForjjDeployMount, req.Forj.ForjjDeploymentEnv) // Must be created by Forjj with git init...
	deployBase := path.Join(deployRepo, req.Forj.ForjjInstanceName)
	deployFile := path.Join(deployBase, github_file)

	// Path in the context of GIT.
	gitFile := path.Join(req.Forj.ForjjInstanceName, github_file)

	if _, err := os.Stat(sourcePath); err != nil {
		if err = os.MkdirAll(sourcePath, 0755); err != nil {
			ret.Errorf("Unable to create '%s'. %s", sourcePath, err)
		}
	}

	if _, err := os.Stat(deployRepo); err != nil {
		ret.Errorf("Unable to create '%s'. Forjj must create it. %s", deployRepo, err)
	}

	if _, err := os.Stat(sourceFile); err == nil {
		ret.Errorf("Unable to create the github configuration which already exist.\nUse 'update' to update it "+
			"(or update %s), and 'maintain' to update your github service according to his configuration.",
			path.Join(instance, github_file))
		return 419
	}

	if _, err := os.Stat(deployBase); err != nil {
		if err = os.Mkdir(deployBase, 0755); err != nil {
			ret.Errorf("Unable to create '%s'. %s", deployBase, err)
		}
	}

	ret.StatusAdd("Environment checked. Ready to be created.")

	// Save gws.github_source.
	if _, err := gws.save_yaml(&gws.github_source, sourceFile); err != nil {
		ret.Errorf("%s", err)
		return
	}
	log.Printf(ret.StatusAdd("Configuration saved in source Repo '%s' (%s).", gitFile, sourceRepo))

	if _, err := gws.save_yaml(&gws.githubDeploy, deployFile); err != nil {
		ret.Errorf("%s", err)
		return
	}
	log.Printf(ret.StatusAdd("Configuration saved in deploy Repo '%s' (%s).", gitFile, deployRepo))

	// Building final Post answer
	// We assume ssh is used and forjj can push with appropriate credential.
	for k, v := range gws.github_source.Urls {
		ret.Services.Urls[k] = v
	}
	// Official application API recognized by Forjj
	ret.Services.Urls["api_url"] = gws.github_source.Urls["github-base-url"]

	ret.CommitMessage = fmt.Sprint("Github configuration created.")
	ret.AddFile(goforjj.FilesSource, gitFile)
	ret.AddFile(goforjj.FilesDeploy, gitFile)

	return
}

// Do updating plugin task
// req_data contains the request data posted by forjj. Structure generated from 'github.yaml'.
// ret_data contains the response structure to return back to forjj.
//
// By default, if httpCode is not set (ie equal to 0), the function caller will set it to 422 in case of errors (error_message != "") or 200
func DoUpdate(w http.ResponseWriter, r *http.Request, req *UpdateReq, ret *goforjj.PluginData) (httpCode int) {
	instance := req.Forj.ForjjInstanceName
	log.Print("Checking Infrastructure code existence.")

	var gws GitHubStruct

	if a, found := req.Objects.App[instance]; !found {
		ret.Errorf("Invalid request. Missing Objects/App/%s", instance)
		return
	} else {
		gws = GitHubStruct{
			source_mount: req.Forj.ForjjSourceMount,
			deployMount:  req.Forj.ForjjDeployMount,
			token:        a.Token,
			app:          &a,
		}
	}

	check := make(map[string]bool)
	check["token"] = true
	check["source"] = true
	log.Printf("Checking parameters : %#v", gws)

	if gws.verify_req_fails(ret, check) {
		return
	}

	confFile := path.Join(gws.deployMount, req.Forj.ForjjDeploymentEnv, instance, github_file)
	if _, err := os.Stat(confFile); err != nil {
		log.Printf(ret.StatusAdd("Warning! The workspace do not contain '%s'", confFile))
		if gws.github_connect(req.Objects.App[instance].Server, ret) == nil {
			return
		}
		if !req.InitOrganization(&gws) {
			log.Printf(ret.Errorf("Unable to update. The organization was not set in the request."))
			return
		}
		gws.req_repos_exists(req, ret)
		ret.Errorf("Unable to update the github configuration which doesn't exist.\n"+
			"Use 'create' to create it "+
			"(or create %s), and 'maintain' to update your github service according to his configuration.",
			path.Join(instance, github_file))
		log.Printf("Unable to update the github configuration '%s'", confFile)
		return 419
	}

	// Read the github.yaml file.
	if err := gws.load_yaml(confFile); err != nil {
		ret.Errorf("Unable to update github instance '%s' source files. %s. Use 'create' to create it first.", instance, err)
		return 419
	}

	if gws.github_connect(req.Objects.App[instance].Server, ret) == nil {
		return
	}

	ret.StatusAdd("Environment checked. Ready to be updated.")

	if _, err := gws.update_yaml_data(req, ret); err != nil {
		ret.Errorf("Unable to update. %s", err)
		return
	}

	// Returns the collection of all managed repository with their existence flag.
	gws.repos_exists(ret)

	// Save gws.github_source.
	if Updated, err := gws.save_yaml(&gws.githubDeploy, confFile); err != nil {
		ret.Errorf("%s", err)
		return
	} else {
		if !Updated {
			log.Printf(ret.StatusAdd("No update detected."))
		} else {
			log.Printf(ret.StatusAdd("Configuration saved in '%s'.", path.Join(instance, github_file)))
			for k, v := range gws.github_source.Urls {
				ret.Services.Urls[k] = v
			}

			ret.CommitMessage = fmt.Sprint("Github configuration updated.")
			ret.AddFile(goforjj.FilesDeploy, path.Join(instance, github_file))
		}
	}

	// Building final Post answer
	// We assume ssh is used and forjj can push with appropriate credential.
	for k, v := range gws.github_source.Urls {
		ret.Services.Urls[k] = v
	}
	// Official application API recognized by Forjj
	ret.Services.Urls["api_url"] = gws.github_source.Urls["github-base-url"]

	return
}

// Do maintaining plugin task
// req_data contains the request data posted by forjj. Structure generated from 'github.yaml'.
// ret_data contains the response structure to return back to forjj.
//
// By default, if httpCode is not set (ie equal to 0), the function caller will set it to 422 in case of errors (error_message != "") or 200
func DoMaintain(w http.ResponseWriter, r *http.Request, req *MaintainReq, ret *goforjj.PluginData) (httpCode int) {
	instance := req.Forj.ForjjInstanceName

	var gws GitHubStruct
	if a, found := req.Objects.App[instance]; !found {
		ret.Errorf("Invalid request. Missing Objects/App/%s", instance)
		return
	} else {
		gws = GitHubStruct{
			deployMount:     req.Forj.ForjjDeployMount,
			workspace_mount: req.Forj.ForjjWorkspaceMount,
			token:           a.Token,
			maintain_ctxt:   true,
			force:           (req.Forj.Force == "true"),
		}
	}
	check := make(map[string]bool)
	check["token"] = true
	check["workspace"] = true
	check["deploy"] = true

	if gws.verify_req_fails(ret, check) { // true => include workspace testing.
		return
	}

	confFile := path.Join(gws.deployMount, req.Forj.ForjjDeploymentEnv, instance, github_file)
	// Read the github.yaml file.
	if err := gws.load_yaml(confFile); err != nil {
		ret.Errorf("%s", err)
		return
	}

	if gws.github_connect("", ret) == nil {
		return
	}

	// ensure organization exist
	if !gws.ensure_organization_exists(ret) {
		return
	}

	if !gws.IsNewForge(ret) {
		return
	}

	if !gws.setOrganizationTeams(ret) {
		return
	}
	log.Printf(ret.StatusAdd("Organization maintained."))

	if !gws.MaintainOrgHooks(ret) {
		return
	}

	if gws.githubDeploy.NoRepos {
		log.Printf(ret.StatusAdd("Repositories maintained limited to your infra repository"))
	}
	// loop on list of repos, and ensure they exist with minimal config and rights
	for name, repo_data := range gws.githubDeploy.Repos {
		if !repo_data.Infra && gws.githubDeploy.NoRepos {
			log.Printf(ret.StatusAdd("Repo ignored: %s", name))
		}
		if err := repo_data.ensure_exists(&gws, ret); err != nil {
			return
		}
		if !gws.MaintainHooks(&repo_data, ret) {
			return
		}
		log.Printf(ret.StatusAdd("Repo maintained: %s", name))
	}
	return
}
