package main

import (
	"flag"
	"fmt"
	"github.com/KrakenSystems/wg-operator/pkg/controller/client"
	"github.com/KrakenSystems/wg-operator/pkg/controller/server"
	"github.com/KrakenSystems/wg-operator/pkg/wgctl"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"os"
	"runtime"

	"github.com/KrakenSystems/wg-operator/pkg/apis"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 6060
)
var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

type logrusLogf struct {
	logger logrus.FieldLogger
}

func (l logrusLogf) Info(msg string, keysAndValues ...interface{}) {
	l.WithValues(keysAndValues...).(logrusLogf).logger.Info(msg)
}

func (l logrusLogf) Enabled() bool {
	return true
}

func (l logrusLogf) Error(err error, msg string, keysAndValues ...interface{}) {
	o := l.WithValues(keysAndValues...).(logrusLogf)
	o.logger.WithError(err).Error(msg)
}

func (l logrusLogf) V(level int) logr.InfoLogger {
	return l
}

func (l logrusLogf) WithValues(keysAndValues ...interface{}) logr.Logger {
	olog := l.logger
	for i := 0; i < len(keysAndValues); i+=2 {
		olog = olog.WithField(fmt.Sprint(keysAndValues[i]), keysAndValues[i + 1])
	}
	return logrusLogf{olog}
}

func (l logrusLogf) WithName(name string) logr.Logger {
	return logrusLogf{l.logger.WithField("name", name)}
}

func main() {
	hostname, err := os.Hostname()
	if err != nil {
		log.Error(err, "cannot retrieve hostname")
		os.Exit(2)
	}

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	mode := pflag.String("mode", "client", "mode the controller is in (server/client)")
	nodeName := pflag.String("node-name", hostname, "hostname")
	interfaceName := pflag.String("wg-interface", "wg0", "interface to configure")
	privateKeyFile := pflag.String("wg-private-key-file", "/etc/wireguard/wg0.key", "wireguard private key file")

	pflag.Parse()

	logf.SetLogger(logrusLogf{logrus.WithField("name", "wg-operator")})
	logrus.SetLevel(logrus.TraceLevel)

	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	wg := &wgctl.WireguardSetup{
		NodeName:       *nodeName,
		InterfaceName:  *interfaceName,
		PrivateKeyFile: *privateKeyFile,
	}
	switch *mode {
	case "client":
		log.Info("Running in client mode", "name", *nodeName)
		if err := client.Add(mgr, wg); err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
	case "server":
		log.Info("Running in server mode", "name", *nodeName)
		if err := server.Add(mgr, wg); err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
	default:
		log.Info("unknown mode: " + *mode)
		os.Exit(5)
	}
	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
