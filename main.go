package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/spf13/cobra"

	"ec2-autorestore/commands"
)

func main() {
	var rootCmd = &cobra.Command{Use: "ec2-restore"}
	var profile string
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "", "AWS profile to use")
	rootCmd.AddCommand(commands.BackupCommand(), commands.RestoreCommand(), commands.PruneCommand())
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		var opts []func(*config.LoadOptions) error
		if profile != "" {
			opts = append(opts, config.WithSharedConfigProfile(profile))
		}

		cfg, err := config.LoadDefaultConfig(context.Background(), opts...)
		if err != nil {
			return err
		}
		ec2c := ec2.NewFromConfig(cfg)

		commands.InitialiseCommands(ec2c)
		return nil
	}
	err := rootCmd.Execute()

	if err != nil {
		log.Fatal(err)
	}
}
