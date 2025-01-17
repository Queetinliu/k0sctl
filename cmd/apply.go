package cmd

import (
	"fmt"
	"time"

	"github.com/k0sproject/k0sctl/analytics"
	"github.com/k0sproject/k0sctl/phase"
	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1"
	log "github.com/sirupsen/logrus"

	"github.com/urfave/cli/v2"
)

var applyCommand = &cli.Command{    //这里使用github.com/urfave/cli/v2这个工具
	Name:  "apply",
	Usage: "Apply a k0sctl configuration",
	Flags: []cli.Flag{
		configFlag,
		&cli.BoolFlag{
			Name:  "no-wait",   //flag名为no-wait   
			Usage: "Do not wait for worker nodes to join",
		},
		&cli.BoolFlag{
			Name:  "no-drain",
			Usage: "Do not drain worker nodes when upgrading",
		},
		&cli.StringFlag{
			Name:      "restore-from",
			Usage:     "Path to cluster backup archive to restore the state from",
			TakesFile: true,
		},
		&cli.BoolFlag{
			Name:   "disable-downgrade-check",
			Usage:  "Skip downgrade check",
			Hidden: true,
		},
		debugFlag,   //这几个参数见cmd/flags.go的定义
		traceFlag,
		redactFlag,
		analyticsFlag,
		upgradeCheckFlag,
	},
	Before: actions(initLogging, startCheckUpgrade, initConfig, displayLogo, initAnalytics, displayCopyright, warnOldCache), 
	//都定义在cmd/flags.go。首先执行这些操作
	After:  actions(reportCheckUpgrade, closeAnalytics),
	Action: func(ctx *cli.Context) error {
		start := time.Now()
		phase.NoWait = ctx.Bool("no-wait")

		manager := phase.Manager{Config: ctx.Context.Value(ctxConfigKey{}).(*v1beta1.Cluster)}  //在这里得到之前获取到的值,context这样传递值比较便捷
		lockPhase := &phase.Lock{}
		//这里AddPhase就运行这些结构体里的Run()方法.这里添加结构体，然后在下面会运行这些结构体的方法
		manager.AddPhase(
			&phase.Connect{},
			&phase.DetectOS{},
			lockPhase,
			&phase.PrepareHosts{},
			&phase.GatherFacts{},
			&phase.DownloadBinaries{},
			&phase.UploadFiles{},
			&phase.ValidateHosts{},
			&phase.GatherK0sFacts{},
			&phase.ValidateFacts{SkipDowngradeCheck: ctx.Bool("disable-downgrade-check")},
			&phase.UploadBinaries{},
			&phase.DownloadK0s{},
			&phase.RunHooks{Stage: "before", Action: "apply"},
			&phase.PrepareArm{},
			&phase.ConfigureK0s{},
			&phase.Restore{
				RestoreFrom: ctx.String("restore-from"),
			},
			&phase.InitializeK0s{},
			&phase.InstallControllers{},
			&phase.InstallWorkers{},
			&phase.UpgradeControllers{},
			&phase.UpgradeWorkers{
				NoDrain: ctx.Bool("no-drain"),
			},
			&phase.RunHooks{Stage: "after", Action: "apply"},
			&phase.Unlock{Cancel: lockPhase.Cancel},
			&phase.Disconnect{},
		)

		analytics.Client.Publish("apply-start", map[string]interface{}{})

		var result error

		if result = manager.Run(); result != nil {  //这里读取manager配置然后顺序执行那些phase
			//每个phase都是用了嵌入GenericPhase，嵌入GenericPhase包含prepare方法，将当前配置给到phase，因此每个phase都能读取到配置
			analytics.Client.Publish("apply-failure", map[string]interface{}{"clusterID": manager.Config.Spec.K0s.Metadata.ClusterID})
			if lf, err := LogFile(); err == nil {
				if ln, ok := lf.(interface{ Name() string }); ok {
					log.Errorf("apply failed - log file saved to %s", ln.Name())
				}
			}
			return result
		}

		analytics.Client.Publish("apply-success", map[string]interface{}{"duration": time.Since(start), "clusterID": manager.Config.Spec.K0s.Metadata.ClusterID})

		duration := time.Since(start).Truncate(time.Second)
		text := fmt.Sprintf("==> Finished in %s", duration)
		log.Infof(Colorize.Green(text).String())

		log.Infof("k0s cluster version %s is now installed", manager.Config.Spec.K0s.Version)
		log.Infof("Tip: To access the cluster you can now fetch the admin kubeconfig using:")
		log.Infof("     " + Colorize.Cyan("k0sctl kubeconfig").String())

		return nil
	},
}
