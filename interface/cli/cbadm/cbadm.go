package main

import (
	"log"

	"github.com/cloud-barista/cb-spider/interface/cli/cbadm/cmd"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// ===== [ Private Functions ] =====

// main - Entrypoint
func main() {
	rootCmd := cmd.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		log.Println("cbadm client terminated with error: ", err.Error())
	}
}

// ===== [ Public Functions ] =====
