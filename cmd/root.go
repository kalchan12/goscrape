package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfgFile      string
	verbose      bool
	logFile      string
	noColor      bool
	logger       *zap.Logger
	sugaredLog   *zap.SugaredLogger
)

var rootCmd = &cobra.Command{
	Use:   "goscrape",
	Short: "GoScrape - terminal web scraper with CLI and TUI modes",
	Long:  `A fast terminal web scraper built in Go with CLI commands and interactive TUI.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initLogger()
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(banner)
		fmt.Print(usageExamples)
		cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.goscrape.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
	rootCmd.PersistentFlags().StringVar(&logFile, "log-file", "", "write logs to file")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colorized output")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("log-file", rootCmd.PersistentFlags().Lookup("log-file"))
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigName(".goscrape")
		viper.SetConfigType("yaml")
	}

	viper.SetDefault("default_output", filepath.Join(os.Getenv("HOME"), "goscrape-output"))
	viper.SetDefault("default_delay", "1s")
	viper.SetDefault("default_workers", 3)
	viper.SetDefault("default_format", "json")
	viper.SetDefault("rotate_agents", true)
	viper.SetDefault("user_agent", "GoScrape/1.0")

	viper.ReadInConfig()
}

func initLogger() error {
	var zapCfg zap.Config
	if verbose {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		zapCfg = zap.NewProductionConfig()
		zapCfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	}

	if noColor {
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	var outputs []string = []string{"stderr"}
	if logFile != "" {
		outputs = append(outputs, logFile)
	}
	zapCfg.OutputPaths = outputs
	zapCfg.ErrorOutputPaths = outputs

	var err error
	logger, err = zapCfg.Build()
	if err != nil {
		return err
	}
	sugaredLog = logger.Sugar()
	return nil
}

func GetLogger() *zap.Logger {
	return logger
}

func GetSugaredLogger() *zap.SugaredLogger {
	return sugaredLog
}

func GetConfigString(key string) string {
	return viper.GetString(key)
}

func GetConfigInt(key string) int {
	return viper.GetInt(key)
}

func GetConfigBool(key string) bool {
	return viper.GetBool(key)
}

func GetDefaultOutputDir() string {
	return viper.GetString("default_output")
}

var banner = "\x1b[1;36m" +
	`  ____      ____` + "\n" +
	` / ___| ___/ ___|  ___ _ __ __ _ _ __   ___` + "\n" +
	`| |  _ / _ \___ \ / __| '__/ _` + "`" + ` | '_ \ / _ \` + "\n" +
	`| |_| | (_) |__) | (__| | | (_| | |_) |  __/` + "\n" +
	` \____|\___/____/ \___|_|  \__,_| .__/ \___|` + "\n" +
	`                                |_|         ` + "\n" +
	"\x1b[0m" + "\x1b[33m" + `           Author: Psycho` + "\n" +
	"\x1b[0m" + "\n"

var usageExamples = "\x1b[1;37mExamples:\x1b[0m\n" +
	"  \x1b[32mgoscrape run --url https://example.com\x1b[0m               Scrape a website\n" +
	"  \x1b[32mgoscrape run --url https://example.com --js\x1b[0m           Scrape with headless browser\n" +
	"  \x1b[32mgoscrape extract --url https://example.com\x1b[0m            Extract JSON-LD, meta, OG tags\n" +
	"  \x1b[32mgoscrape download --url https://example.com --types pdf,json\x1b[0m  Download files\n" +
	"  \x1b[32mgoscrape history list\x1b[0m                                  View crawl history\n" +
	"  \x1b[32mgoscrape interactive\x1b[0m                                   Launch TUI mode\n\n"

func init() {
	// suppress extra init output
}