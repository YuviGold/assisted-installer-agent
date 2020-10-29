package main

import (
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/commands"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func main() {
	config.ProcessArgs()
	util.SetLogging("agent", config.GlobalAgentConfig.TextLogging, config.GlobalAgentConfig.JournalLogging)
	if config.GlobalAgentConfig.IsText {
		o, _, _ := commands.GetInventory("")
		fmt.Print(o)
	} else if config.GlobalAgentConfig.ConnectivityParams != "" {
		output, errStr, exitCode := commands.ConnectivityCheck("", config.GlobalAgentConfig.ConnectivityParams)
		if exitCode != 0 {
			fmt.Println(errStr)
		} else {
			fmt.Println(output)
		}
	} else {
		output, errStr, exitCode := commands.StartNTPDaemon()
		if exitCode != 0 {
			fmt.Println(errStr)
			os.Exit(exitCode)
		} else {
			fmt.Println(output)
		}

		commands.RegisterHostWithRetry()
		commands.ProcessSteps()
	}
}
