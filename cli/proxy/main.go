package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/laurentsimon/jupyter-lineage/cli/proxy/internal/logger"
	"github.com/laurentsimon/jupyter-lineage/cli/proxy/internal/repository"
	"github.com/laurentsimon/jupyter-lineage/cli/proxy/internal/utils"
	"github.com/laurentsimon/jupyter-lineage/pkg/jnproxy"
	"github.com/laurentsimon/jupyter-lineage/pkg/slsa"
)

func usage(prog string) {
	msg := "" +
		"Usage: %s srcIP, srcShellPort, srcStdinPort, srcIOPubPort, srcControlPort, srcHeartBeatPort\n" +
		"dstIP, dstShellPort, dstStdinPort, dstIOPubPort, dstControlPort, dstHeartBeatPort\n" +
		"provenancePath, certDir"
	utils.Log(msg, prog)
	os.Exit(1)
}

func fatal(e error) {
	utils.Log("error: %v\n", e)
	os.Exit(2)
}

func main() {
	arguments := os.Args[1:]
	if len(arguments) != 14 {
		usage(os.Args[0])
	}

	// src metadata.
	srcIP := arguments[0]
	srcShellPort := arguments[1]
	srcStdinPort := arguments[2]
	srcIOPubPort := arguments[3]
	srcControlPort := arguments[4]
	srcHeartbeatPort := arguments[5]
	// dst metadata.
	dstIP := arguments[6]
	dstShellPort := arguments[7]
	dstStdinPort := arguments[8]
	dstIOPubPort := arguments[9]
	dstControlPort := arguments[10]
	dstHeartbeatPort := arguments[11]
	repoDir := arguments[12]
	certDir := arguments[13]

	utils.Log("%q %q %q %q %q %q %q %q %q %q %q %q %q\n",
		srcIP, srcShellPort, srcStdinPort, srcIOPubPort, srcControlPort, srcHeartbeatPort,
		dstIP, dstShellPort, dstStdinPort, dstIOPubPort, dstControlPort, dstHeartbeatPort,
		certDir,
	)

	jserverConfig, err := jnproxy.JServerConfigNew(
		jnproxy.NetworkConfig{
			IP: srcIP,
			Ports: jnproxy.Ports{
				Shell:     utils.StringToUint(srcShellPort),
				Stdin:     utils.StringToUint(srcStdinPort),
				IOPub:     utils.StringToUint(srcIOPubPort),
				Control:   utils.StringToUint(srcControlPort),
				Heartbeat: utils.StringToUint(srcHeartbeatPort),
			},
		},
		jnproxy.NetworkConfig{
			IP: dstIP,
			Ports: jnproxy.Ports{
				Shell:     utils.StringToUint(dstShellPort),
				Stdin:     utils.StringToUint(dstStdinPort),
				IOPub:     utils.StringToUint(dstIOPubPort),
				Control:   utils.StringToUint(dstControlPort),
				Heartbeat: utils.StringToUint(dstHeartbeatPort),
			},
		},
	)
	if err != nil {
		fatal(fmt.Errorf("JServerConfigNew: %w", err))
	}

	httpConfig, err := jnproxy.HttpConfigNew([]string{"localhost:9999"})
	if err != nil {
		fatal(fmt.Errorf("HttpConfigNew: %w", err))
	}

	// Create our logger.
	workingDir, err := os.Getwd()
	if err != nil {
		fatal(fmt.Errorf("get working directory: %w", err))
	}
	logFn := filepath.Join(workingDir, "proxy.log")
	if _, err := os.Stat(logFn); err == nil {
		os.Remove(logFn)
	}
	f, err := os.Create(logFn)
	if err != nil {
		fatal(fmt.Errorf("get working directory: %w", err))
	}
	defer f.Close()

	opts := []logger.Option{logger.WithWriter(f)}
	//opts := []logger.Option{}
	logger, err := logger.New(opts...)
	if err != nil {
		fatal(fmt.Errorf("logger new: %w", err))
	}

	// Create repo client.
	os.RemoveAll(repoDir)
	if err := os.MkdirAll(repoDir, os.ModePerm); err != nil {
		fatal(fmt.Errorf("mkdir: %w", err))
	}
	repoClient, err := repository.New(logger, repoDir)
	if err != nil {
		logger.Fatalf("create repo client: %v", err)
	}
	// Read CA
	cert, err := os.Open(filepath.Join(certDir, "ca.cert"))
	if err != nil {
		fatal(fmt.Errorf("read cert: %w", err))
	}
	key, err := os.Open(filepath.Join(certDir, "ca.key"))
	if err != nil {
		fatal(fmt.Errorf("read key: %w", err))
	}
	// Create a new jnproxy.
	proxy, err := jnproxy.New(*jserverConfig, *httpConfig,
		repoClient, jnproxy.WithLogger(logger),
		jnproxy.WithCA(jnproxy.CA{Certificate: cert, Key: key}),
		jnproxy.InstallHuggingfaceModel(),
		jnproxy.InstallDenyHandler())
	if err != nil {
		logger.Fatalf("create proxy: %v", err)
	}
	// Start the jnproxy.
	if err := proxy.Start(); err != nil {
		logger.Fatalf("start proxy: %v", err)
	}

	// os.Kill?
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-c
		if err := proxy.Stop(); err != nil {
			logger.Fatalf("stop proxy: %v", err)
		}
		subjects := []slsa.Subject{
			{
				Name: "modelX",
				DigestSet: slsa.DigestSet{
					"sha256": "86cbdad53be99e43661bbbd2f22d95680334d92d579404a4747b1d15373da263",
				},
			},
		}
		prov, err := proxy.Provenance(slsa.Builder{ID: "https://colab.googleapis.com/ColabHostedKernel"}, subjects, "")
		if err != nil {
			logger.Fatalf("provenance: %v", err)
		}
		logger.Infof("prov: %s", prov)
		if err := os.WriteFile(filepath.Join(repoDir, "prov.json"), prov, 0644); err != nil {
			logger.Fatalf("write provenance: %v", err)
		}
		logger.Infof("Exiting...\n")
		os.Exit(0)
	}()

	for {

	}
}
