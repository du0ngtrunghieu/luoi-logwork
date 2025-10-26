/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/du0ngtrunghieu/luoi-logwork/cmd/internal/configure"
	"github.com/du0ngtrunghieu/luoi-logwork/cmd/internal/logwork"
	"github.com/du0ngtrunghieu/luoi-logwork/pkg/types"
	"github.com/spf13/cobra"
)

// logworkCmd represents the logwork command
var logworkCmd = &cobra.Command{
	Use:   "logwork",
	Short: "Auto logwork",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		execute()
	},
}

var estimateCmd = &cobra.Command{
	Use:   "est",
	Short: "Auto fill estimate",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		executeEstimate()
	},
}

func execute() {
	config := &types.Config{}
	configure.ReadConfig(config)

	var projectTracking logwork.ProjectTracking

	switch config.EndpointType {
	case "jira":
		projectTracking = logwork.NewJira(config.Endpoint, config.Username, config.ApiToken)
	default:
		fmt.Println("Endpoint type not supported")
	}

	tickets, err := projectTracking.GetTicketToLog()
	if err != nil {
		fmt.Println(err)
		return
	}

	dayToLog, err := projectTracking.GetDayToLog()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = projectTracking.LogWork(tickets, dayToLog)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func executeEstimate() {
	config := &types.Config{}
	configure.ReadConfig(config)
	var projectTracking logwork.ProjectTracking

	switch config.EndpointType {
	case "jira":
		projectTracking = logwork.NewJira(config.Endpoint, config.Username, config.ApiToken)
	default:
		fmt.Println("Endpoint type not supported")
	}

	tickets, err := projectTracking.GetTicketToEst()
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := projectTracking.AddEstForTicket(tickets); err != nil {
		fmt.Println("Error updating estimates:", err)
	}

}
func init() {
	rootCmd.AddCommand(logworkCmd)
	rootCmd.AddCommand(estimateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// logworkCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// logworkCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
