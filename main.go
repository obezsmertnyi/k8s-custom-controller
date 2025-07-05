/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/obezsmertnyi/k8s-custom-controller/cmd"
	"github.com/rs/zerolog/log"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Error().Err(err).Msg("CLI execution failed")
	}
}
