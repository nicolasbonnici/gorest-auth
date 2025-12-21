package main

import (
	"log"

	"github.com/nicolasbonnici/gorest"
	"github.com/nicolasbonnici/gorest/pluginloader"

	authplugin "github.com/nicolasbonnici/gorest-auth"
)

func init() {
	pluginloader.RegisterPluginFactory("auth", authplugin.NewPlugin)
}

func main() {
	cfg := gorest.Config{
		ConfigPath: ".",
	}

	log.Println("Starting GoREST with Auth Plugin...")
	gorest.Start(cfg)
}
