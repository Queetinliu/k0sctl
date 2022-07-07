package cmd

import (
	"github.com/urfave/cli/v2"
)

// App is the main urfave/cli.App for k0sctl
var App = &cli.App{
	Name:  "k0sctl",
	Usage: "k0s cluster management tool",
	Flags: []cli.Flag{
		debugFlag,
		traceFlag,
		redactFlag,
	},
	Commands: []*cli.Command{
		versionCommand,  //见cmd/version.go，输出kubectl版本
		applyCommand, //cmd/apply.go，apply命令
		kubeconfigCommand,
		initCommand,
		resetCommand,
		backupCommand,
		{
			Name:  "config",
			Usage: "Configuration related sub-commands",
			Subcommands: []*cli.Command{
				configEditCommand,
				configStatusCommand,
			},
		},
		completionCommand,
	},
	EnableBashCompletion: true,
}   //使用上面提到的cli服务，生成命令。
