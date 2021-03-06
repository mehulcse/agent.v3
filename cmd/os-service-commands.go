package cmd

import (
	"github.com/pinpt/agent/cmd/cmdserviceinstall"
	"github.com/pinpt/agent/cmd/cmdservicerestart"
	"github.com/pinpt/agent/cmd/cmdserviceruninternal"
	"github.com/pinpt/agent/cmd/cmdservicestart"
	"github.com/pinpt/agent/cmd/cmdservicestatus"
	"github.com/pinpt/agent/cmd/cmdservicestop"
	"github.com/pinpt/agent/cmd/cmdserviceuninstall"
	"github.com/pinpt/agent/cmd/pkg/cmdlogger"
	"github.com/spf13/cobra"
)

var cmdServiceRunInternal = &cobra.Command{
	Use:   "service-run-internal",
	Short: "Run agent service. This is called automatically once the service is installed.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// set to debug log output from restarter, it will not affect lower level components
		logger := cmdlogger.NewLoggerJSON(cmd, "debug")

		pinpointRoot, err := getPinpointRoot(cmd)
		if err != nil {
			exitWithErr(logger, err)
		}

		err = cmdserviceruninternal.Run(logger, pinpointRoot)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmd := cmdServiceRunInternal
	cmd.Hidden = true
	flagPinpointRoot(cmd)
	cmdRoot.AddCommand(cmd)
}

var cmdServiceInstall = &cobra.Command{
	Use:   "service-install",
	Short: "Install agent as OS managed service",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// set to debug log output from restarter, it will not affect lower level components
		logger := cmdlogger.NewLoggerJSON(cmd, "debug")

		pinpointRoot, err := getPinpointRoot(cmd)
		if err != nil {
			exitWithErr(logger, err)
		}

		start, _ := cmd.Flags().GetBool("start")
		err = cmdserviceinstall.Run(logger, pinpointRoot, start)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmd := cmdServiceInstall
	flagPinpointRoot(cmd)
	cmd.Flags().Bool("start", false, "start the service after install")
	cmdRoot.AddCommand(cmd)
}

var cmdServiceStart = &cobra.Command{
	Use:   "service-start",
	Short: "Start agent service",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// set to debug log output from restarter, it will not affect lower level components
		logger := cmdlogger.NewLoggerJSON(cmd, "debug")
		err := cmdservicestart.Run(logger)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmd := cmdServiceStart
	cmdRoot.AddCommand(cmd)
}

var cmdServiceStop = &cobra.Command{
	Use:   "service-stop",
	Short: "Stop agent service",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// set to debug log output from restarter, it will not affect lower level components
		logger := cmdlogger.NewLoggerJSON(cmd, "debug")
		err := cmdservicestop.Run(logger)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmd := cmdServiceStop
	cmdRoot.AddCommand(cmd)
}

var cmdServiceStatus = &cobra.Command{
	Use:   "service-status",
	Short: "Show status of agent service",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// set to debug log output from restarter, it will not affect lower level components
		logger := cmdlogger.NewLoggerJSON(cmd, "debug")
		err := cmdservicestatus.Run(logger)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmd := cmdServiceStatus
	cmdRoot.AddCommand(cmd)
}

var cmdServiceRestart = &cobra.Command{
	Use:   "service-restart",
	Short: "Restart agent service",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// set to debug log output from restarter, it will not affect lower level components
		logger := cmdlogger.NewLoggerJSON(cmd, "debug")
		err := cmdservicerestart.Run(logger)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmdRoot.AddCommand(cmdServiceRestart)
}

var cmdServiceUninstall = &cobra.Command{
	Use:   "service-uninstall",
	Short: "Uninstall OS service of agent, but keep data and configuration",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// set to debug log output from restarter, it will not affect lower level components
		logger := cmdlogger.NewLoggerJSON(cmd, "debug")
		err := cmdserviceuninstall.Run(logger)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmdRoot.AddCommand(cmdServiceUninstall)
}
