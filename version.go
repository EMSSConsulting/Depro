package main

import "github.com/EMSSConsulting/Depro/version"

var (
	GitCommit   string
	GitDescribe string
)

const Version = "1.0.0"

const VersionPrerelease = "dev"

func init() {
	version.Version = Version
	version.Release = VersionPrerelease
	version.GitCommit = GitCommit

	if GitDescribe != "" {
		version.Version = GitDescribe
	}

	if GitDescribe == "" && version.Release == "" {
		version.Release = "dev"
	}
}
