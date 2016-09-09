package main

import (
    "gopkg.in/yaml.v2"
    "fmt"
    "io/ioutil"
    "github.hpe.com/christophe-larsonneur/goforjj"
    "reflect"
)

func (g *GitHubStruct)create_yaml_data(req *CreateReq) error {
    // Write the github.yaml source file.
    g.github_source.Urls = make(map[string]string)
    g.github_source.Urls["github-base-url"] = g.Client.BaseURL.String()

    req.InitOrganization(g)

    if g.github_source.Repos == nil {
        g.github_source.Repos = make(map[string]RepositoryStruct)
    }

    for name, repo := range req.ReposData {
        g.AddRepo(name, repo)
    }

    // TODO: Be able to add several repos thanks to the request structure.
    return nil
}

// Add a new repository to be managed by github plugin.
func (g *GitHubStruct)AddRepo(name string, repo goforjj.PluginRepoData) bool{
    upstream := "git@" + g.Client.BaseURL.Host + ":" + g.github_source.Organization + "/" + name + ".git"

    if r, found := g.github_source.Repos[name] ; ! found {
        r = RepositoryStruct{
            Description: repo.Title,
            Users: repo.Users,
            Groups: repo.Groups,
            Flow: repo.Flow,
            Name: name,
            remotes: map[string]string {"origin":upstream},
            branchConnect: map[string]string {"master":"origin/master"},
        }
        g.github_source.Repos[name] = r
        return true // New added
    }
    return false
}

func (r *RepositoryStruct)Update(repo goforjj.PluginRepoData) (count int){
    if r.Description != repo.Title {
        r.Description = repo.Title
        count++
    }

    if r.Flow != repo.Flow {
        r.Flow = repo.Flow
        count++
    }

    if ! reflect.DeepEqual(r.Users, repo.Users) {
        r.Users = repo.Users
        count++
    }
    if !reflect.DeepEqual(r.Groups, repo.Groups) {
        r.Groups = repo.Groups
        count++
    }
    return
}

func (g *GitHubStruct)save_yaml(file string) error {

    d, err := yaml.Marshal(&g.github_source)
    if  err != nil {
        return fmt.Errorf("Unable to encode github data in yaml. %s", err)
    }

    if err := ioutil.WriteFile(file, d, 0644) ; err != nil {
        return fmt.Errorf("Unable to save '%s'. %s", file, err)
    }
    return nil
}

func (g *GitHubStruct)load_yaml(file string) error {
    d, err := ioutil.ReadFile(file)
    if err != nil {
        return fmt.Errorf("Unable to load '%s'. %s", file, err)
    }

    err = yaml.Unmarshal(d, &g.github_source)
    if  err != nil {
        return fmt.Errorf("Unable to decode github data in yaml. %s", err)
    }
    return nil
}
