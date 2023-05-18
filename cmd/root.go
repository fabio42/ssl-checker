package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fabio42/ssl-checker/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configFile string
	envCheck   string
)

var rootCmd = &cobra.Command{
	Use:   "ssl-checker [flags] [files <files>|domains <domains>]",
	Short: "ssl-checker",
	Long:  "ssl-checker is a tool to _quickly_ check certificate details of multiple https targets.",

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := setLogger(viper.GetBool("debug"))
		if err != nil {
			log.Fatal().Msgf("Error failed to configure logger:", err)
		}

		if viper.GetBool("debug") {
			log.Warn().Msgf("Debug is enabled, log will be found in %v", logFile)
		}

		cfgFile := filepath.Base(configFile)
		cfgPath := filepath.Dir(configFile)
		viper.SetConfigName(cfgFile[:len(cfgFile)-len(filepath.Ext(cfgFile))])
		viper.AddConfigPath(cfgPath)
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				log.Debug().Msg("No config file found")
			} else {
				log.Fatal().Msgf("Error while parsing config: %v", err)
			}
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		envQuery := strings.Split(envCheck, ",")

		if viper.IsSet("queries") {
			queries := viper.Get("queries")
			fileTargets := map[string]string{}
			domainTargets := map[string][]string{}

			for env, data := range queries.(map[string]interface{}) {
				if envCheck != "" && !sliceContains(envQuery, env) {
					continue
				}
				switch data := data.(type) {
				case string:
					fileTargets[env] = data
				case []interface{}:
					domains := make([]string, len(data))
					for k, domain := range data {
						switch domain := domain.(type) {
						case string:
							domains[k] = domain
						default:
							log.Fatal().Msgf("Unsupported data type in query option for %s: %v is of type %T", env, domain, domain)
						}
						domainTargets[env] = domains
					}
				default:
					log.Fatal().Msgf("Unsupported data type in queries option: %v is of type %T", data, data)
				}
			}
			log.Debug().Msgf("fileTargets is  : %v", fileTargets)
			log.Debug().Msgf("domainTargets is: %v", domainTargets)

			runQueries(fileTargets, domainTargets)

		} else {
			// Nothing to do
			log.Debug().Msgf("Empty query... nothing to do")
			os.Exit(1)
		}
	},
}

var listEnvs = &cobra.Command{
	Use:   "environments",
	Short: "List environments set in configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		if viper.IsSet("queries") {
			queries := viper.Get("queries")
			envs := make([]string, len(queries.(map[string]interface{})))

			i := 0
			for env := range queries.(map[string]interface{}) {
				envs[i] = env
				i++
			}
			fmt.Printf("Available environments: %s.\n", strings.Join(envs, ", "))
		} else {
			// Nothing to do
			log.Debug().Msgf("Empty query... nothing to do")
			os.Exit(1)
		}
	},
}

func runQueries(fileTargets map[string]string, domainTargets map[string][]string) {
	if viper.GetBool("silent") {
		fmt.Fprintln(os.Stderr, "Processing query!")
	}
	q := ui.NewModel(viper.GetInt("timeout"), viper.GetBool("silent"), fileTargets, domainTargets)
	if err := tea.NewProgram(q).Start(); err != nil {
		log.Fatal().Msgf("Error while running TUI program: %v", err)
	}
}

func sliceContains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "$HOME/.config/ssl-checker/config.yaml", "Configuration file location")
	rootCmd.PersistentFlags().BoolP("silent", "s", false, "disable ui")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug log, out will be saved in "+logFile)
	rootCmd.PersistentFlags().Uint16P("timeout", "t", 10, "Set timeout for SSL check queries")
	rootCmd.Flags().StringVarP(&envCheck, "environments", "e", "", "Comma delimited string specifying the environments to check")

	viper.BindPFlag("silent", rootCmd.PersistentFlags().Lookup("silent"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))

	rootCmd.AddCommand(listEnvs)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Msgf("Whoops. There was an error while executing your CLI '%s'", err)
	}
}
