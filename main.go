package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mylxsw/s3-cli/internal"
)

//go:embed config.yaml.example
var defaultConfigTemplate []byte

func main() {
	configPathFlag := flag.String("config", "", "Path to config file (default: ~/.s3-cli/config.yaml)")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	uniqueFlag := flag.Bool("unique", false, "Use random filename instead of content hash")
	serverNameFlag := flag.String("server", "", "Server name defined in config")
	initConfigFlag := flag.Bool("init-config", false, "Create default config file at config path and exit")
	flag.Parse()

	internal.SetDebug(*debugFlag)

	configPath := *configPathFlag
	if configPath == "" {
		path, err := internal.DefaultConfigPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to resolve config path:", err)
			os.Exit(1)
		}
		configPath = path
	}

	if *initConfigFlag {
		if err := writeDefaultConfig(configPath); err != nil {
			fmt.Fprintln(os.Stderr, "failed to write config:", err)
			os.Exit(1)
		}
		fmt.Println("config written to", configPath)
		return
	}

	if *debugFlag {
		fmt.Fprintln(os.Stderr, "debug: using config", configPath)
	}

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: s3-cli [options] <image-path> [more-paths...]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	imagePaths := flag.Args()
	for _, imagePath := range imagePaths {
		if err := validateFilePath(imagePath); err != nil {
			fmt.Fprintln(os.Stderr, "invalid image path:", err)
			os.Exit(1)
		}
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

	for _, imagePath := range imagePaths {
		_, url, err := internal.UploadFile(context.Background(), server, imagePath, *uniqueFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "upload failed:", err)
			os.Exit(1)
		}
		fmt.Println(url)
	}
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

func writeDefaultConfig(path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists at %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, defaultConfigTemplate, 0o644)
}
