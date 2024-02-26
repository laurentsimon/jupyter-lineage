package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/laurentsimon/jupyter-lineage/cli/proxy/internal/utils"
	"github.com/laurentsimon/jupyter-lineage/pkg/session"
)

func usage(prog string) {
	msg := "" +
		"Usage: %s listeningIP, listeningShellPort, listeningStdinPort, listeningIOPubPort, listeningControlPort, listeningHeartBeatPort\n" +
		"destinationIP, destinationShellPort, destinationStdinPort, destinationIOPubPort, destinationControlPort, destinationHeartBeatPort\n"
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
	// Listening metadata.
	listeningIP := arguments[0]
	listeningShellPort := arguments[1]
	listeningStdinPort := arguments[2]
	listeningIOPubPort := arguments[3]
	listeningControlPort := arguments[4]
	listeningHeartbeatPort := arguments[5]
	// Destination metadata.
	destinationIP := arguments[6]
	destinationShellPort := arguments[7]
	destinationStdinPort := arguments[8]
	destinationIOPubPort := arguments[9]
	destinationControlPort := arguments[10]
	destinationHeartbeatPort := arguments[11]

	utils.Log("%q %q %q %q %q %q %q %q %q %q %q %q\n",
		listeningIP, listeningShellPort, listeningStdinPort, listeningIOPubPort, listeningControlPort, listeningHeartbeatPort,
		destinationIP, destinationShellPort, destinationStdinPort, destinationIOPubPort, destinationControlPort, destinationHeartbeatPort,
	)

	listeningMetadata := session.NetworkMetadata{
		IP: listeningIP,
		Ports: session.Ports{
			Shell:     utils.StringToUint(listeningShellPort),
			Stdin:     utils.StringToUint(listeningStdinPort),
			IOPub:     utils.StringToUint(listeningIOPubPort),
			Control:   utils.StringToUint(listeningControlPort),
			Heartbeat: utils.StringToUint(listeningHeartbeatPort),
		},
	}
	destinationMetadata := session.NetworkMetadata{
		IP: destinationIP,
		Ports: session.Ports{
			Shell:     utils.StringToUint(destinationShellPort),
			Stdin:     utils.StringToUint(destinationStdinPort),
			IOPub:     utils.StringToUint(destinationIOPubPort),
			Control:   utils.StringToUint(destinationControlPort),
			Heartbeat: utils.StringToUint(destinationHeartbeatPort),
		},
	}
	workingDir, err := os.Getwd()
	if err != nil {
		fatal(fmt.Errorf("get working directory: %w", err))
	}
	repoDir := filepath.Join(workingDir, "jupyter_repo")
	opts := []session.Option{session.WithRepositoryDir(repoDir)}
	session, err := session.New(listeningMetadata, destinationMetadata, opts...)
	if err != nil {
		fatal(fmt.Errorf("create session: %w", err))
	}

	if err := session.Start(); err != nil {
		fatal(fmt.Errorf("start session: %w", err))
	}

	if err := session.End(); err != nil {
		fatal(fmt.Errorf("end session: %w", err))
	}

	utils.Log("Exiting...\n")
	os.Exit(0)
}
