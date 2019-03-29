package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/KrakenSystems/wg-operator/pkg/apis"
	"github.com/KrakenSystems/wg-operator/pkg/controller/node"
	"github.com/KrakenSystems/wg-operator/pkg/logrAdapter"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag" // _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost = "0.0.0.0"
)
var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	hostname, err := os.Hostname()
	if err != nil {
		log.Error(err, "cannot retrieve hostname")
		os.Exit(2)
	}

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	mode := pflag.String("mode", "client", "mode the controller is in (server/client)")
	nodeName := pflag.String("node-name", hostname, "hostname")
	iface := pflag.String("wg-interface", "wg0", "interface to configure")
	privateKeyFile := pflag.String("wg-private-key-file", "/etc/wireguard/wg0.key", "wireguard private key file")
	metricsPort := pflag.Int("metrics-port", 6060, "metrics port")
	metric := pflag.Int("route-metric", 100, "metric to use for routing table")
	proto := pflag.Int("route-proto", 121, "daemon route table protocol number")
	table := pflag.Int("route-table", 0, "daemon route table number")
	dryRun := pflag.BoolP("dry-run", "n", false, "Dry run")
	syncConfigPath := pflag.String("sync-config-path", "/etc/wireguard", "Config file sync location. PATH/<<iface>>.conf")
	syncConfig := pflag.Bool("sync-config", false, "whether to sync config files")
	splitServers := pflag.Bool("split-servers", false, "create interface per server. Highly experimental")

	pflag.Parse()

	logf.SetLogger(logrAdapter.NewLogrusAdapter(logrus.WithField("name", "wg-operator")))
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
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, *metricsPort),
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

	ctlCfg := node.NodeControllerConfig{
		NodeName:       *nodeName,
		Interface:      *iface,
		PrivateKeyFile: *privateKeyFile,
		Namespace:      namespace,
		RouteMetric:    *metric,
		RouteProto:     *proto,
		RouteTable:     *table,
		DryRun:         *dryRun,
		SyncConfigPath: *syncConfigPath,
		SyncConfig:     *syncConfig,
		SplitServers:   *splitServers,
	}

	switch *mode {
	case "client":
		log.Info("Running in client mode", "name", *nodeName)
		ctlCfg.Mode = node.Client
	case "server":
		log.Info("Running in server mode", "name", *nodeName)
		ctlCfg.Mode = node.Server
	default:
		log.Info("unknown mode: " + *mode)
		os.Exit(5)
	}

	if err := node.Add(mgr, ctlCfg); err != nil {
		log.Error(err, "Cannot add node controller")
		os.Exit(6)
	}
	log.Info("Starting the Cmd.")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
