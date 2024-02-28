package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/laurentsimon/jupyter-lineage/cli/proxy/internal/logger"
	"github.com/laurentsimon/jupyter-lineage/cli/proxy/internal/repository"
	"github.com/laurentsimon/jupyter-lineage/cli/proxy/internal/utils"
	"github.com/laurentsimon/jupyter-lineage/pkg/session"
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
	if len(arguments) != 12 {
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
	logger := logger.Logger{}
	// Create repo client.
	repoClient, err := repository.New(logger)
	if err != nil {
		fatal(fmt.Errorf("create session: %w", err))
	}
	// Create a new session.
	session, err := session.New(srcMetadata, dstMetadata,
		session.WithLogger(logger),
		session.WithRepositoryClient(repoClient))
	if err != nil {
		fatal(fmt.Errorf("create session: %w", err))
	}
	// Start the session.
	if err := session.Start(); err != nil {
		fatal(fmt.Errorf("start session: %w", err))
	}

	// os.Kill?
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-c
		if err := session.Finish(); err != nil {
			fatal(fmt.Errorf("finish session: %w", err))
		}
		utils.Log("Exiting...\n")
		os.Exit(0)
	}()

	for {

	}
}
