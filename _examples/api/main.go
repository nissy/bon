package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	defaultCfgName = "api.conf"
	version        = "0.1"
)

var (
	cfgName   = flag.String("c", defaultCfgName, "set cfgiguration file")
	isHelp    = flag.Bool("h", false, "this help")
	isVersion = flag.Bool("v", false, "show this build version")
)

func main() {
	os.Exit(exitcode(run()))
}

func exitcode(err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return 1
	}

	return 0
}

func run() error {
	flag.Parse()

	if *isVersion {
		fmt.Println("v" + version)
		return nil
	}

	if *isHelp {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
		return nil
	}

	sv := newService()

	if err := sv.applyConfig(*cfgName); err != nil {
		return err
	}

	if err := sv.serve(); err != nil {
		return err
	}

	return nil
}
