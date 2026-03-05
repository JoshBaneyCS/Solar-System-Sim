package main

import (
	"flag"
	"fmt"
	"os"

	"solar-system-sim/internal/assets"
)

func main() {
	dir := flag.String("dir", "assets", "Asset directory path")
	flag.Parse()

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
