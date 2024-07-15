package main

import (
	"os"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	phpfpm "github.com/initializ-buildpacks/php-fpm"
)

func main() {
	logEmitter := scribe.NewEmitter(os.Stdout).WithLevel(os.Getenv("BP_LOG_LEVEL"))
	config := phpfpm.NewConfig()
	packit.Run(phpfpm.Detect(), phpfpm.Build(config, logEmitter))
}
