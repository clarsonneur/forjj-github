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
	"path"

	"github.com/forj-oss/goforjj"
)

const github_file = "github.yaml"

// DoCreate creating plugin task
// req_data contains the request data posted by forjj. Structure generated from 'github.yaml'.
// ret_data contains the response structure to return back to forjj.
//
// By default, if httpCode is not set (ie equal to 0), the function caller will set it to 422 in case of errors (error_message != "") or 200
func DoCreate(w http.ResponseWriter, r *http.Request, req *CreateReq, ret *goforjj.PluginData) (httpCode int) {
	instance := req.Forj.ForjjInstanceName
	gws := GitHubStruct{
		source_mount: req.Forj.ForjjSourceMount,
		deployMount:  req.Forj.ForjjDeployMount,
		instance:     instance,
		deployTo:     req.Forj.ForjjDeploymentEnv,
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

	if !req.InitOrganization(&gws) {
		ret.Errorf("Internal Error. Unable to define the organization")
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

	if err := gws.checkSourcesExistence("create"); err != nil {
		ret.Errorf("%s\nUnable to 'create' your forge", err)
		return
	}

	ret.StatusAdd("Environment checked. Ready to be created.")

	// Path in the context of GIT.
	gitFile := path.Join(gws.instance, github_file)

	// Save gws.github_source.
	if _, err := gws.save_yaml(&gws.github_source, gws.sourceFile); err != nil {
		ret.Errorf("%s", err)
		return
	}
	log.Printf(ret.StatusAdd("Configuration saved in source Repo '%s' (%s).", gitFile, gws.source_mount))

	if _, err := gws.save_yaml(&gws.githubDeploy, gws.deployFile); err != nil {
		ret.Errorf("%s", err)
		return
	}
	log.Printf(ret.StatusAdd("Configuration saved in deploy Repo '%s' (%s).", gitFile, path.Join(gws.deployMount, gws.deployTo)))

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
			instance:     instance,
			deployTo:     req.Forj.ForjjDeploymentEnv,
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

	if err := gws.checkSourcesExistence("update"); err != nil {
		ret.Errorf("%s\nUnable to 'update' your forge", err)
		return
	}

	// Read the github.yaml file.
	if err := gws.load_yaml(gws.sourceFile); err != nil {
		ret.Errorf("Unable to update github instance '%s' source files. %s. Use 'create' to create it first.", instance, err)
		return 419
	}

	if !req.InitOrganization(&gws) {
		log.Printf(ret.Errorf("Unable to update. The organization was not set in the request."))
		return
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
	if Updated, err := gws.save_yaml(&gws.github_source, gws.sourceFile); err != nil {
		ret.Errorf("%s", err)
		return
	} else {
		if !Updated {
			log.Printf(ret.StatusAdd("Source: No github configuration update detected."))
		} else {
			log.Printf(ret.StatusAdd("Source: github configuration saved in '%s'.", path.Join(instance, github_file)))

			ret.CommitMessage = fmt.Sprint("Source: github configuration updated.")
			ret.AddFile(goforjj.FilesSource, path.Join(instance, github_file))
		}
	}

	// Save gws.github_deploy.
	if Updated, err := gws.save_yaml(&gws.githubDeploy, gws.deployFile); err != nil {
		ret.Errorf("%s", err)
		return
	} else {
		if !Updated {
			log.Printf(ret.StatusAdd("Deploy: No github configuration update detected."))
		} else {
			log.Printf(ret.StatusAdd("Deploy: github configuration saved in '%s'.", path.Join(instance, github_file)))

			ret.CommitMessage = fmt.Sprint("Deploy: github configuration updated.")
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
			continue
		}
		if repo_data.Role == "infra" && !repo_data.IsDeployable {
			log.Printf(ret.StatusAdd("Repo ignored: %s - Infra repo owned by '%s'", name, gws.githubDeploy.ProdOrganization))
			continue
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
