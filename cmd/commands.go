package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/pinpt/agent/cmd/cmdenroll"
	"github.com/pinpt/agent/cmd/cmdexport"
	"github.com/pinpt/agent/cmd/cmdexportonboarddata"
	"github.com/pinpt/agent/cmd/cmdservicerun"
	"github.com/pinpt/agent/cmd/cmdservicerunnorestarts"
	"github.com/pinpt/agent/cmd/cmdvalidate"
	"github.com/pinpt/agent/cmd/cmdvalidateconfig"
	"github.com/pinpt/agent/cmd/pkg/cmdlogger"
	"github.com/pinpt/agent/rpcdef"
	pos "github.com/pinpt/go-common/os"
	"github.com/spf13/cobra"
)

func isInsideDocker() bool {
	return pos.IsInsideContainer()
}

var cmdEnrollNoServiceRun = &cobra.Command{
	Use:   "enroll-no-service-run <code>",
	Short: "Enroll the agent with the Pinpoint Cloud",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// only json is supported as log format for enroll and service-run, since it proxies the logs from subcommands, from which export is required to be json to be sent to the server corretly
		cmd.Flags().Set("log-format", "json")

		code := args[0]
		logger := cmdlogger.NewLogger(cmd)
		pinpointRoot, err := getPinpointRoot(cmd)
		if err != nil {
			exitWithErr(logger, err)
		}

		// once we have pinpoint root, we can also log to a file
		logWriter, err := pinpointLogWriter(pinpointRoot)
		if err != nil {
			exitWithErr(logger, err)
		}
		logger = logger.AddWriter(logWriter)

		channel, _ := cmd.Flags().GetString("channel")
		ctx := context.Background()
		skipValidate, _ := cmd.Flags().GetBool("skip-validate")

		if !skipValidate {
			valid, err := cmdvalidate.Run(ctx, logger, pinpointRoot)
			if err != nil {
				exitWithErr(logger, err)
			}
			if !valid {
				os.Exit(1)
			}
		}

		integrationsDir, _ := cmd.Flags().GetString("integrations-dir")
		skipEnroll, _ := cmd.Flags().GetBool("skip-enroll-if-found")

		err = cmdenroll.Run(ctx, cmdenroll.Opts{
			Logger:            logger,
			PinpointRoot:      pinpointRoot,
			IntegrationsDir:   integrationsDir,
			Code:              code,
			Channel:           channel,
			SkipEnrollIfFound: skipEnroll,
		})
		if err != nil {
			exitWithErr(logger, err)
		}

		logger.Info("enroll command completed")
	},
}

func init() {
	cmd := cmdEnrollNoServiceRun
	flagsLogger(cmd)
	flagPinpointRoot(cmd)
	cmd.Flags().String("integrations-dir", defaultIntegrationsDir(), "Integrations dir")
	cmd.Flags().String("channel", "stable", "Cloud channel to use.")
	cmd.Flags().Bool("skip-validate", false, "skip minimum requirements")
	cmd.Flags().Bool("skip-enroll-if-found", false, "skip enroll if the config is already found")
	cmdRoot.AddCommand(cmd)
}

var cmdEnroll = &cobra.Command{
	Use:   "enroll <code>",
	Short: "Enroll the agent with the Pinpoint Cloud",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		skipServiceRun, _ := cmd.Flags().GetBool("skip-service-run")
		if skipServiceRun {
			cmdEnrollNoServiceRun.Run(cmd, args)
			return
		}

		// only json is supported as log format for service-run, since it proxies the logs from subcommands, from which export is required to be json to be sent to the server corretly
		cmd.Flags().Set("log-format", "json")

		logger := cmdlogger.NewLogger(cmd)
		pinpointRoot, err := getPinpointRoot(cmd)
		if err != nil {
			exitWithErr(logger, err)
		}

		skipValidate, _ := cmd.Flags().GetBool("skip-validate")
		skipEnroll, _ := cmd.Flags().GetBool("skip-enroll-if-found")
		channel, _ := cmd.Flags().GetString("channel")
		integrationsDir, _ := cmd.Flags().GetString("integrations-dir")

		ctx := context.Background()
		opts := cmdservicerun.Opts{}
		opts.Logger = logger
		opts.PinpointRoot = pinpointRoot
		opts.IntegrationsDir = integrationsDir
		opts.Enroll.Run = true
		opts.Enroll.Code = args[0]
		opts.Enroll.Channel = channel
		opts.Enroll.SkipValidate = skipValidate
		opts.Enroll.SkipEnrollIfFound = skipEnroll
		err = cmdservicerun.Run(ctx, opts)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmd := cmdEnroll
	flagsLogger(cmd)
	flagPinpointRoot(cmd)
	cmd.Flags().String("integrations-dir", defaultIntegrationsDir(), "Integrations dir")
	cmd.Flags().String("channel", "stable", "Cloud channel to use.")
	cmd.Flags().Bool("skip-validate", false, "skip minimum requirements")
	cmd.Flags().Bool("skip-service-run", false, "Set to true to skip service run. Will need to run it separately.")
	cmd.Flags().Bool("skip-enroll-if-found", false, "skip enroll if the config is already found")
	cmdRoot.AddCommand(cmd)
}

var cmdExport = &cobra.Command{
	Use:    "export",
	Hidden: true,
	Short:  "Export all data of multiple passed integrations",
	Args:   cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		opts := cmdexport.Opts{}
		logger, opts2 := integrationCommandOpts(cmd)
		opts.Opts = opts2
		opts.ReprocessHistorical, _ = cmd.Flags().GetBool("reprocess-historical")

		outputFile := newOutputFile(logger, cmd)
		defer outputFile.Close()
		opts.Output = outputFile.Writer

		err := cmdexport.Run(opts)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmd := cmdExport
	integrationCommandFlags(cmd)
	flagOutputFile(cmd)
	cmd.Flags().Bool("reprocess-historical", false, "Set to true to discard incremental checkpoint and reprocess historical instead.")
	cmdRoot.AddCommand(cmd)
}

var cmdValidateConfig = &cobra.Command{
	Use:    "validate-config",
	Hidden: true,
	Short:  "Validates the configuration by making a test connection",
	Args:   cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		logger, baseOpts := integrationCommandOpts(cmd)
		opts := cmdvalidateconfig.Opts{}
		opts.Opts = baseOpts

		outputFile := newOutputFile(logger, cmd)
		defer outputFile.Close()
		opts.Output = outputFile.Writer

		err := cmdvalidateconfig.Run(opts)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmd := cmdValidateConfig
	integrationCommandFlags(cmd)
	flagOutputFile(cmd)
	cmdRoot.AddCommand(cmd)
}

var cmdExportOnboardData = &cobra.Command{
	Use:    "export-onboard-data",
	Hidden: true,
	Short:  "Exports users, repos or projects based on param for a specified integration. Saves that data into provided file.",
	Args:   cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		logger, baseOpts := integrationCommandOpts(cmd)
		opts := cmdexportonboarddata.Opts{}
		opts.Opts = baseOpts

		outputFile := newOutputFile(logger, cmd)
		defer outputFile.Close()
		opts.Output = outputFile.Writer

		{
			v, _ := cmd.Flags().GetString("object-type")
			if v == "" {
				exitWithErr(logger, errors.New("provide object-type arg"))
			}
			if v == "users" || v == "repos" || v == "projects" || v == "workconfig" {
				opts.ExportType = rpcdef.OnboardExportType(v)
			} else {
				exitWithErr(logger, fmt.Errorf("object-type must be one of: users, repos, projects, got %v", v))
			}
		}

		err := cmdexportonboarddata.Run(opts)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmd := cmdExportOnboardData
	integrationCommandFlags(cmd)
	flagOutputFile(cmd)
	cmd.Flags().String("object-type", "", "Object type to export, one of: users, repos, projects.")
	cmdRoot.AddCommand(cmd)
}

/*
var cmdServiceInstall = &cobra.Command{
	Use:   "service-install",
	Short: "Install OS service of agent",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		logger := cmdlogger.NewLogger(cmd)
		err := cmdserviceinstall.Run(logger)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmdRoot.AddCommand(cmdServiceInstall)
}

var cmdServiceUninstall = &cobra.Command{
	Use:   "service-uninstall",
	Short: "Uninstall OS service of agent, but keep data and configuration",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		logger := cmdlogger.NewLogger(cmd)
		err := cmdserviceuninstall.Run(logger)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmdRoot.AddCommand(cmdServiceUninstall)
}*/

var cmdServiceRunNoRestarts = &cobra.Command{
	Use:   "service-run-no-restarts",
	Short: "This command is called by OS service to run the service.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {

		// only json is supported as log format for service-run, since it proxies the logs from subcommands, from which export is required to be json to be sent to the server corretly
		cmd.Flags().Set("log-format", "json")
		logger := cmdlogger.NewLogger(cmd)
		pinpointRoot, err := getPinpointRoot(cmd)
		if err != nil {
			exitWithErr(logger, err)
		}
		logWriter, err := pinpointLogWriter(pinpointRoot)
		if err != nil {
			exitWithErr(logger, err)
		}
		logger = logger.AddWriter(logWriter)

		ctx := context.Background()
		opts := cmdservicerunnorestarts.Opts{}
		opts.Logger = logger
		opts.LogLevelSubcommands = logger.Level
		opts.PinpointRoot = pinpointRoot
		err = cmdservicerunnorestarts.Run(ctx, opts)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmd := cmdServiceRunNoRestarts
	flagsLogger(cmd)
	flagPinpointRoot(cmd)
	cmdRoot.AddCommand(cmd)
}

var cmdServiceRun = &cobra.Command{
	Use:   "service-run",
	Short: "This command is called by OS service to run the service.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// only json is supported as log format for service-run, since it proxies the logs from subcommands, from which export is required to be json to be sent to the server corretly
		cmd.Flags().Set("log-format", "json")

		logger := cmdlogger.NewLogger(cmd)
		pinpointRoot, err := getPinpointRoot(cmd)
		if err != nil {
			exitWithErr(logger, err)
		}

		ctx := context.Background()
		opts := cmdservicerun.Opts{}
		opts.Logger = logger
		opts.PinpointRoot = pinpointRoot
		err = cmdservicerun.Run(ctx, opts)
		if err != nil {
			exitWithErr(logger, err)
		}
	},
}

func init() {
	cmd := cmdServiceRun
	flagsLogger(cmd)
	flagPinpointRoot(cmd)
	cmdRoot.AddCommand(cmd)
}

var cmdVersion = &cobra.Command{
	Use:   "version",
	Short: "Display the build version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:", Version)
		fmt.Println("Commit:", Commit)
	},
}

func init() {
	cmdRoot.AddCommand(cmdVersion)
}

var cmdValidate = &cobra.Command{
	Use:   "validate",
	Short: "Validate minimum hardware requirements",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {

		ctx := context.Background()
		logger := cmdlogger.NewLogger(cmd)
		pinpointRoot, err := getPinpointRoot(cmd)
		if err != nil {
			exitWithErr(logger, err)
		}

		if _, err := cmdvalidate.Run(ctx, logger, pinpointRoot); err != nil {
			exitWithErr(logger, err)
		}

	},
}

func init() {
	cmd := cmdValidate
	integrationCommandFlags(cmd)
	cmdRoot.AddCommand(cmd)
}
