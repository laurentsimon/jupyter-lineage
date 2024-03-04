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
	"github.com/laurentsimon/jupyter-lineage/pkg/session"
	"github.com/laurentsimon/jupyter-lineage/pkg/slsa"
)

func usage(prog string) {
	msg := "" +
		"Usage: %s srcIP, srcShellPort, srcStdinPort, srcIOPubPort, srcControlPort, srcHeartBeatPort\n" +
		"dstIP, dstShellPort, dstStdinPort, dstIOPubPort, dstControlPort, dstHeartBeatPort\n"
	utils.Log(msg, prog)
	os.Exit(1)
}

func fatal(e error) {
	utils.Log("error: %v\n", e)
	os.Exit(2)
}

func main() {
	arguments := os.Args[1:]
	if len(arguments) != 13 {
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

	utils.Log("%q %q %q %q %q %q %q %q %q %q %q %q\n",
		srcIP, srcShellPort, srcStdinPort, srcIOPubPort, srcControlPort, srcHeartbeatPort,
		dstIP, dstShellPort, dstStdinPort, dstIOPubPort, dstControlPort, dstHeartbeatPort,
	)

	srcMetadata := session.NetworkMetadata{
		IP: srcIP,
		Ports: session.Ports{
			Shell:     utils.StringToUint(srcShellPort),
			Stdin:     utils.StringToUint(srcStdinPort),
			IOPub:     utils.StringToUint(srcIOPubPort),
			Control:   utils.StringToUint(srcControlPort),
			Heartbeat: utils.StringToUint(srcHeartbeatPort),
		},
	}
	dstMetadata := session.NetworkMetadata{
		IP: dstIP,
		Ports: session.Ports{
			Shell:     utils.StringToUint(dstShellPort),
			Stdin:     utils.StringToUint(dstStdinPort),
			IOPub:     utils.StringToUint(dstIOPubPort),
			Control:   utils.StringToUint(dstControlPort),
			Heartbeat: utils.StringToUint(dstHeartbeatPort),
		},
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
	// Create a new session.
	session, err := session.New(srcMetadata, dstMetadata,
		repoClient, session.WithLogger(logger))
	if err != nil {
		logger.Fatalf("create session: %v", err)
	}
	// Start the session.
	if err := session.Start(); err != nil {
		logger.Fatalf("start session: %v", err)
	}

	// os.Kill?
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-c
		if err := session.Stop(); err != nil {
			logger.Fatalf("stop session: %v", err)
		}
		subjects := []slsa.Subject{
			{
				Name: "modelX",
				DigestSet: slsa.DigestSet{
					"sha1": "86cbdad53be99e43661bbbd2f22d95680334d92d579404a4747b1d15373da263",
				},
			},
		}
		prov, err := session.Provenance(slsa.Builder{ID: "my-builder-id"}, subjects, "")
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
