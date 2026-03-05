package main

import (
	"flag"
	"fmt"
	"os"

	"solar-system-sim/internal/assets"
)

func runAssets(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: solar-sim assets <subcommand>")
		fmt.Fprintln(os.Stderr, "  verify   Verify asset directory structure")
		os.Exit(1)
	}

	switch args[0] {
	case "verify":
		runAssetsVerify(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown assets subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func runAssetsVerify(args []string) {
	fs := flag.NewFlagSet("assets verify", flag.ExitOnError)
	dir := fs.String("dir", "assets", "Asset directory path")
	fs.Parse(args)

	errors, infos := assets.ValidateAssets(*dir)

	for _, info := range infos {
		fmt.Printf("INFO: %s\n", info)
	}

	if len(errors) > 0 {
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", e)
		}
		os.Exit(1)
	}

	fmt.Printf("All assets validated successfully (%d body textures, models, credits)\n", assets.BodyCount())
}
