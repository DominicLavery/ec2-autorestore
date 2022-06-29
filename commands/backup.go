package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/cobra"
)

func BackupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "backup [tagValue] [backupID]",
		Short: "Creates a snapshot of instances",
		Long:  `Creates a snapshot of instances that have the tag backup with a value of [tagValue] and gives it an ID of [backupID]`,
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			createSnapshots(args[0], args[1])
		},
	}
}

func createSnapshots(tagValue string, backupId string) {
	filters := []types.Filter{{Name: aws.String("tag:backup"), Values: []string{tagValue}}}
	input := &ec2.DescribeInstancesInput{Filters: filters}
	instancesMap := make(map[string]types.Instance)

	for ok := true; ok; {
		out, err := ec2c.DescribeInstances(context.TODO(), input)
		if err != nil {
			log.Fatal("Can't get instances", err)
		}
		for _, r := range out.Reservations {
			for _, i := range r.Instances {
				log.Printf("Found instance \"%v\"", *i.InstanceId)
				instancesMap[*i.InstanceId] = i
			}
		}
		if ok = out.NextToken != nil; ok {
			input.NextToken = out.NextToken
		}
	}

	snapshots, err := getSnapshots(backupId)
	if err != nil {
		log.Fatalf("Error checking for existing backups: %v", err)
	}
	if len(snapshots) > 0 {
		fmt.Printf("There are existing backups with the id \"%v\":\n", backupId)
		err = checkAndDeleteSnaps(snapshots)
		if err != nil {
			log.Fatalf("Could not delete snapshots because %v", err)
		}
	}

	ids := shutdownInstances(instancesMap)

	for id, i := range instancesMap {
		for _, b := range i.BlockDeviceMappings {
			if *b.DeviceName == *i.RootDeviceName {
				log.Printf("Snapshotting %s of %s", id, *b.Ebs.VolumeId)
				tags := append(i.Tags,
					types.Tag{Key: aws.String("autorestore-backupId"), Value: aws.String(backupId)},
					types.Tag{Key: aws.String("autorestore-instanceId"), Value: aws.String(id)},
				)
				csi := &ec2.CreateSnapshotInput{
					VolumeId: b.Ebs.VolumeId,
					TagSpecifications: []types.TagSpecification{{
						ResourceType: "snapshot",
						Tags:         tags,
					}},
				}
				snap, err := ec2c.CreateSnapshot(context.TODO(), csi)
				if err != nil {
					log.Fatal("Failed to create snapshot", err)
				}
				log.Printf("Created %s\n", *snap.SnapshotId)
			}
		}
	}

	restartInstances(ids)
}
