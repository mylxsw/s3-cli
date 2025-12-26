package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mylxsw/s3-cli/internal"
)

func main() {
	configPathFlag := flag.String("config", "", "Path to config file (default: ~/.s3-cli/config.yaml)")
	serverNameFlag := flag.String("server", "", "Server name defined in config")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: s3-cli [options] <image-path>")
		flag.PrintDefaults()
		os.Exit(2)
	}

	imagePath := flag.Arg(0)
	if err := validateFilePath(imagePath); err != nil {
		fmt.Fprintln(os.Stderr, "invalid image path:", err)
		os.Exit(1)
	}

	configPath := *configPathFlag
	if configPath == "" {
		path, err := internal.DefaultConfigPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to resolve config path:", err)
			os.Exit(1)
		}
		configPath = path
	}

	cfg, err := internal.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config:", err)
		os.Exit(1)
	}

	serverName, server, err := cfg.SelectServer(*serverNameFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := server.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid server %q: %v\n", serverName, err)
		os.Exit(1)
	}

	_, url, err := internal.UploadFile(context.Background(), server, imagePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "upload failed:", err)
		os.Exit(1)
	}

	fmt.Println(url)
}

func validateFilePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	return nil
}
