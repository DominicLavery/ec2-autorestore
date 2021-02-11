package main

import (
	"ec2-autorestore/commands"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
	"log"
)

func main() {
	sess := session.Must(session.NewSession(new(aws.Config).WithRegion("eu-west-2")))
	creds := stscreds.NewCredentials(sess, "arn:aws:iam::347316231722:role/fromCore")
	ec2c := ec2.New(sess, &aws.Config{Credentials: creds})

	commands.InitialiseCommands(ec2c)
	var rootCmd = &cobra.Command{Use: "app"}
	rootCmd.AddCommand(commands.BackupCommand(), commands.RestoreCommand())
	err := rootCmd.Execute()

	if err != nil {
		log.Fatal(err)
	}
}
