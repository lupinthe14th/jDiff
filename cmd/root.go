package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	files   []string
	debug   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "jdiff",
	Short: "jdiff is compare json files for command line program.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires a json files arguments")
		}
		if len(args) > 2 {
			return errors.New("specify two json files")
		}
		files = args
		return nil
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Debug().Msgf("files: %v", files)
		src, err := f2b(files[0])
		if err != nil {
			return fmt.Errorf("failed src:%s %v", files[0], err)
		}
		dist, err := f2b(files[1])
		if err != nil {
			return fmt.Errorf("failed dist:%s %v", files[1], err)
		}

		// transformJSON transforms any Go string that looks like JSON into
		// a generic data structure that represents that JSON input.
		// We use an AcyclicTransformer to avoid having the transformer
		// apply on outputs of itself (lest we get stuck in infinite recursion).
		transformJSON := cmpopts.AcyclicTransformer("TransformJSON", func(b []byte) interface{} {
			var v interface{}
			if err := json.Unmarshal(b, &v); err != nil {
				return b // use unparseable input as the output
			}
			return v
		})

		if diff := cmp.Diff(src, dist, transformJSON); diff != "" {
			fmt.Printf("mismatch (-dist +src):\n%s", diff)
		}
		return nil
	},
}

func f2b(s string) ([]byte, error) {
	log.Debug().Msgf("file: %v", s)
	b, err := os.ReadFile(s)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug mode")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jdiff.yaml)")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".jdiff")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
