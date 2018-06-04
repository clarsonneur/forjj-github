// This file has been created by "go generate" as initial code and HAS been updated. Do not remove it.

package main

import (
	"fmt"
	"github.com/forj-oss/goforjj"
	"log"
)

func (g *GitHubStruct) update_yaml_data(req *UpdateReq, ret *goforjj.PluginData) (bool, error) {
	if g.github_source.Urls == nil {
		return false, fmt.Errorf("Internal Error. Urls was not set.")
	}

	if g.githubDeploy.Repos == nil {
		g.githubDeploy.Repos = make(map[string]RepositoryStruct)
	}

	// In update, we simply rebuild Users and Team from Forjfile.
	// No need to keep track of removed one
	g.githubDeploy.Users = make(map[string]string)
	g.githubDeploy.Groups = make(map[string]TeamStruct)

	if g.app.ReposDisabled == "true" {
		log.Print("Repos_disabled is true. forjj_github won't manage repositories except the infra one.")
		g.githubDeploy.NoRepos = true
	} else {
		// Updating all from Forjfile repos
		g.githubDeploy.NoRepos = false
		g.SetOrgHooks(g.app.OrganizationWebhooksDisabled, g.app.ReposWebhooksDisabled, g.app.OrgHookPolicy, req.Objects.Webhooks)
		for name, repo := range req.Objects.Repo {
			if !repo.IsValid(name, ret) {
				continue
			}

			g.SetRepo(&repo, (name == g.app.ForjjInfra), repo.Deployable == "true")
			g.SetHooks(&repo, req.Objects.Webhooks)
		}


		// Disabling missing one
		for name, repo := range g.githubDeploy.Repos {
			if err := repo.IsValid(name); err != nil {
				delete(g.githubDeploy.Repos, name)
				ret.StatusAdd("Warning!!! Invalid repository '%s' found in github.yaml. Removed.")
				continue
			}
			if _, found := req.Objects.Repo[name]; !found && !repo.Disabled {
				repo.Disabled = true
				g.githubDeploy.Repos[name] = repo
				ret.StatusAdd("Disabling repository '%s'", name)
			}
		}
	}

	log.Printf("Github manage %d repository(ies)", len(g.githubDeploy.Repos))

	if g.app.TeamsDisabled == "true" {
		log.Print("Teams_disabled is true. forjj_github won't manage Organization Users.")
		g.githubDeploy.NoTeams = true
	} else {
		g.githubDeploy.NoTeams = false
		for name, details := range req.Objects.User {
			g.AddUser(name, &details)
		}
	}

	log.Printf("Github manage %d user(s) at Organization level.", len(g.githubDeploy.Users))

	if g.githubDeploy.NoTeams {
		log.Print("Teams_disabled is true. forjj_github won't manage Organization Groups.")
	} else {
		for name, details := range req.Objects.Group {
			g.AddGroup(name, &details)
		}
	}

	log.Printf("Github manage %d group(s) at Organization level.", len(g.githubDeploy.Groups))

	return true, nil
}

// SetRepo Add a new repository to be managed by github plugin.
func (g *GitHubStruct) SetRepo(repo *RepoInstanceStruct, isInfra, isDeployable bool) {
	upstream := g.DefineRepoUrls(repo.Name)

	owner := g.githubDeploy.Organization
	if isInfra {
		owner = g.githubDeploy.ProdOrganization
	}

	// found or not, I need to set it.
	r := RepositoryStruct{}
	r.set(repo,
		map[string]goforjj.PluginRepoRemoteUrl{"origin": upstream},
		map[string]string{"master": "origin/master"},
		isInfra, isDeployable, owner)
	g.githubDeploy.Repos[repo.Name] = r

}

// SaveMaintainOptions Function which adds maintain options as part of the plugin answer in create/update phase.
// forjj won't add any driver name because 'maintain' phase read the list of drivers to use from forjj-maintain.yml
// So --git-us is not available for forjj maintain.
func (r *UpdateArgReq) SaveMaintainOptions(ret *goforjj.PluginData) {
	if ret.Options == nil {
		ret.Options = make(map[string]goforjj.PluginOption)
	}
}

func addMaintainOptionValue(options map[string]goforjj.PluginOption, option, value, defaultv, help string) goforjj.PluginOption {
	opt, ok := options[option]
	if ok && value != "" {
		opt.Value = value
		return opt
	}
	if !ok {
		opt = goforjj.PluginOption{Help: help}
		if value == "" {
			opt.Value = defaultv
		} else {
			opt.Value = value
		}
	}
	return opt
}
