package ping

import (
	"fmt"
	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "ping example",
	Long:  `this is ping example`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("run hugo...")
	},
}
