package main

import (
	"os"
	"path/filepath"

	"github.com/bogdanfinn/tls-client-api/internal/tls-client-api/api"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/application"
)

func main() {
	ex, _ := os.Executable()
	configFilePath := filepath.Join(filepath.Dir(ex), "config.dist.yml")

	application.Run(
		application.WithConfigFile(configFilePath, "yml"),
		application.WithConfigFileFlag,
		application.WithModuleFactory("tls-client-api", apiserver.New(api.DefineRouter)),
	)
}
