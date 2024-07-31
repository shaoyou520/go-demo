package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "qt-demo",
	Short: "qt demo is a example",
	Long:  `this is learn example`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("run hugo...")
	},
}

func Execute() {
	rootCmd.AddCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
