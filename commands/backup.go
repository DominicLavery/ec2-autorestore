package commands

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
	"log"
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
	filters := []*ec2.Filter{new(ec2.Filter).SetName("tag:backup").SetValues([]*string{aws.String(tagValue)})}
	input := new(ec2.DescribeInstancesInput).SetFilters(filters)
	var instances []*ec2.Instance
	for ok := true; ok; {
		out, err := ec2c.DescribeInstances(input)
		if err != nil {
			log.Fatal("Can't get instances", err)
		}
		for _, r := range out.Reservations {
			for _, i := range r.Instances {
				for _, b := range i.BlockDeviceMappings {
					if *b.DeviceName == rootVol {
						instances = append(instances, i)
					}
				}
			}
		}
		if ok = out.NextToken != nil; ok {
			input.SetNextToken(*out.NextToken)
		}
	}

	for _, i := range instances {
		log.Printf("Shutting down %s\n", *i.InstanceId)
		_, err := ec2c.StopInstances(new(ec2.StopInstancesInput).SetInstanceIds([]*string{i.InstanceId}))
		if err != nil {
			log.Fatal("Can't stop instance", err)
		}
	}

	err := ec2c.WaitUntilInstanceStopped(input)
	if err != nil {
		log.Fatal("Can't stop instance", err)
	}
	for _, i := range instances {
		for _, b := range i.BlockDeviceMappings {
			if *b.DeviceName == rootVol {
				log.Printf("Snapshotting %s of %s ", *i.InstanceId, *b.Ebs.VolumeId)
				tags := append(i.Tags, new(ec2.Tag).SetKey("autorestore-backupId").SetValue(backupId))
				tags = append(tags, new(ec2.Tag).SetKey("autorestore-instanceId").SetValue(*i.InstanceId))
				csi := new(ec2.CreateSnapshotInput).
					SetVolumeId(*b.Ebs.VolumeId).
					SetTagSpecifications([]*ec2.TagSpecification{new(ec2.TagSpecification).
						SetResourceType("snapshot").
						SetTags(tags)})
				snap, err := ec2c.CreateSnapshot(csi)
				if err != nil {
					log.Fatal("Failed to create snapshot", err)
				}
				log.Printf("Created %s\n", *snap.SnapshotId)
			}
		}
		_, err := ec2c.StartInstances(new(ec2.StartInstancesInput).SetInstanceIds([]*string{i.InstanceId}))
		if err != nil {
			log.Println("Can't start instance", err)
		}
	}
}
