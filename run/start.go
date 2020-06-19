package run

import (
	cmn "github.com/gagarinchain/common"
	golog "github.com/ipfs/go-log"
	"github.com/op/go-logging"
	"os"
)

var stdoutLogFormat = logging.MustStringFormatter(
	`%{time:15:04:05.000} [%{shortfile}] [%{level}] %{message}`,
)

var log = logging.MustGetLogger("main")

const WelcomeArt = `
 ██████╗  █████╗  ██████╗  █████╗ ██████╗ ██╗███╗   ██╗   ███╗   ██╗███████╗████████╗██╗    ██╗ ██████╗ ██████╗ ██╗  ██╗
██╔════╝ ██╔══██╗██╔════╝ ██╔══██╗██╔══██╗██║████╗  ██║   ████╗  ██║██╔════╝╚══██╔══╝██║    ██║██╔═══██╗██╔══██╗██║ ██╔╝
██║  ███╗███████║██║  ███╗███████║██████╔╝██║██╔██╗ ██║   ██╔██╗ ██║█████╗     ██║   ██║ █╗ ██║██║   ██║██████╔╝█████╔╝ 
██║   ██║██╔══██║██║   ██║██╔══██║██╔══██╗██║██║╚██╗██║   ██║╚██╗██║██╔══╝     ██║   ██║███╗██║██║   ██║██╔══██╗██╔═██╗ 
╚██████╔╝██║  ██║╚██████╔╝██║  ██║██║  ██║██║██║ ╚████║██╗██║ ╚████║███████╗   ██║   ╚███╔███╔╝╚██████╔╝██║  ██║██║  ██╗
 ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝    ╚══╝╚══╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝
`

func Start(s *cmn.Settings) {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	initLogger(s.Log.Level)

	log.Info(WelcomeArt)

	ctx := CreateContext(s)

	// Ok now we can bootstrap the node. This could take a little bit if we're
	// running on a live network.
	ctx.Bootstrap(s)
	log.Info("All services bootstrapped successfully")
	select {}

}

func initLogger(logLevel string) {
	level, _ := golog.LevelFromString(logLevel)
	golog.SetAllLoggers(level)

	backend := logging.NewLogBackend(os.Stdout, "", 0)
	errBackend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, stdoutLogFormat)
	errBackendFormatter := logging.NewBackendFormatter(errBackend, stdoutLogFormat)
	backendLeveled := logging.AddModuleLevel(errBackendFormatter)
	errBackendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.INFO, "")
	errBackendLeveled.SetLevel(logging.ERROR, "")

	logging.SetBackend(backendLeveled, backendFormatter)
	logging.SetBackend(errBackendLeveled, errBackendFormatter)
}
