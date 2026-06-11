package cmd

import (
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
	Short: "GoScrape - Terminal-based web scraping tool",
	Long:  `A production-ready terminal web scraper with CLI and TUI modes.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initLogger()
	},
	Run: func(cmd *cobra.Command, args []string) {
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
	viper.SetDefault("python_path", "python3")

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

func init() {
	// suppress extra init output
}