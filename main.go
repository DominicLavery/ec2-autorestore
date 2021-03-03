package main

import (
	"context"
	"ec2-autorestore/commands"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/spf13/cobra"
	"log"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	ec2c := ec2.NewFromConfig(cfg)

	commands.InitialiseCommands(ec2c)
	var rootCmd = &cobra.Command{Use: "ec2-restore"}
	rootCmd.AddCommand(commands.BackupCommand(), commands.RestoreCommand(), commands.PruneCommand())
	err = rootCmd.Execute()

	if err != nil {
		log.Fatal(err)
	}
}
