// Command sample-project serves the SDMS Teacher Management API (FEAT-001).
//
// Startup order matters and is the whole of this file:
//
//  1. load config (the config package does it in its init, so a bad
//     environment stops the process before anything else runs).
//  2. build the composition root - connections, validators, DAOs, services.
//  3. create the router and give it to the HTTP controller.
//  4. register routes.
//  5. generate the swagger document from the routes just registered.
//  6. serve.
//
// The database schema is not applied here. sql_migrations/01_init_schema.sql is
// mounted into the Postgres container's init directory by docker-compose.yml,
// so a fresh volume comes up with the schema already in place. Applying it from
// the application would mean a migration dependency and a second startup path
// for the same file.
package main

import (
	"github.com/nexsoft-git/nexcommon/docs/swagger"
	"github.com/nexsoft-git/nexlogger/log"

	"sample-project/config"
	"sample-project/router"
	"sample-project/server_config"
)

func main() {
	appConfig := config.AppConfig

	serverAttribute := server_config.NewServerAttribute(appConfig)
	err := serverAttribute.Init()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("error found when init server attribute")
	}

	router.InitHttpService(
		appConfig,
		serverAttribute.MetricMiddleware,
		serverAttribute.Validator.HttpController,
		serverAttribute.HealthChecker,
	)

	serverAttribute.InitEndpoint()

	swagger.SwaggerContainer.AppVersion(appConfig.Server.Version)
	swagger.SwaggerContainer.DocumentTitle("SDMS Teacher Management Documentation")
	swagger.SwaggerContainer.DocumentDescription(
		"Teacher CRUD, including education history. No authentication (architecture A2).",
	)

	router.StartService(appConfig, serverAttribute.Validator.HttpController)
}
