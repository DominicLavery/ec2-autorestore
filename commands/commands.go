package commands

import (
	"bufio"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"log"
	"os"
	"strings"
	"time"
)

var ec2c *ec2.Client

var rmVolumes bool

func InitialiseCommands(e *ec2.Client) {
	ec2c = e
}

func shutdownInstances(instancesMap map[string]types.Instance) []string {
	var ids []string
	log.Println("Shutting down instances")
	for _, i := range instancesMap {
		ids = append(ids, *i.InstanceId)
	}

	input := &ec2.StopInstancesInput{InstanceIds: ids}

	_, err := ec2c.StopInstances(context.TODO(), input)
	if err != nil {
		log.Fatal("Can't stop instance", err)
	}
	waiter := ec2.NewInstanceStoppedWaiter(ec2c)
	maxWaitTime := 5 * time.Minute
	params := &ec2.DescribeInstancesInput{InstanceIds: ids}
	err = waiter.Wait(context.TODO(), params, maxWaitTime)
	if err != nil {
		log.Fatal("Can't stop instance", err)
	}
	return ids
}

func restartInstances(ids []string) {
	log.Printf("Restarting instances")

	_, err := ec2c.StartInstances(context.TODO(), &ec2.StartInstancesInput{InstanceIds: ids})
	if err != nil {
		log.Println("Can't start instance", err)
	}
}

func getSnapshots(backupId string) ([]types.Snapshot, error) {
	var snapshots []types.Snapshot
	filters := []types.Filter{{Name: aws.String("tag:autorestore-backupId"), Values: []string{backupId}}}
	var ownerIds = []string{"self"}
	input := &ec2.DescribeSnapshotsInput{OwnerIds: ownerIds, Filters: filters}
	for ok := true; ok; {
		out, err := ec2c.DescribeSnapshots(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, out.Snapshots...)
		if ok = out.NextToken != nil; ok {
			input.NextToken = out.NextToken
		}
	}
	return snapshots, nil
}

func checkAndDeleteSnaps(snaps []types.Snapshot) error {
	fmt.Println("Are you sure you would like to delete the following snapshots?")
	for _, s := range snaps {
		fmt.Printf("%v ", *s.SnapshotId)
	}
	fmt.Println()
	fmt.Println()

	for ok := false; !ok; {
		fmt.Printf("Delete or cancel [d/c]? ")
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		text = strings.TrimSuffix(text, "\n")
		switch text {
		case "c":
			return new(UserCancelError)
		case "d":
			ok = true
		}
	}
	return deleteSnapshots(snaps)
}

func deleteSnapshots(snaps []types.Snapshot) error {
	for _, s := range snaps {
		log.Printf("Deleting snapshot \"%v\"", *s.SnapshotId)
		_, err := ec2c.DeleteSnapshot(context.TODO(), &ec2.DeleteSnapshotInput{SnapshotId: s.SnapshotId})
		if err != nil {
			return err
		}
	}
	return nil
}

func getVolumes(backupId string) ([]types.Volume, error) {
	var vols []types.Volume
	filters := []types.Filter{
		{Name: aws.String("tag:autorestore-backupId"), Values: []string{backupId}},
		{Name: aws.String("status"), Values: []string{"available"}},
	}
	input := &ec2.DescribeVolumesInput{Filters: filters}
	for ok := true; ok; {
		out, err := ec2c.DescribeVolumes(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		vols = append(vols, out.Volumes...)
		if ok = out.NextToken != nil; ok {
			input.NextToken = out.NextToken
		}
	}
	return vols, nil
}

func getVolumesByIds(volIds []string) ([]types.Volume, error) {
	var vols []types.Volume
	input := &ec2.DescribeVolumesInput{VolumeIds: volIds}
	for ok := true; ok; {
		out, err := ec2c.DescribeVolumes(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		vols = append(vols, out.Volumes...)
		if ok = out.NextToken != nil; ok {
			input.NextToken = out.NextToken
		}
	}
	return vols, nil
}

func deleteVolumes(vols []types.Volume) error {
	for _, v := range vols {
		log.Printf("Deleting volume \"%v\"", *v.VolumeId)
		_, err := ec2c.DeleteVolume(context.TODO(), &ec2.DeleteVolumeInput{VolumeId: v.VolumeId})
		if err != nil {
			return err
		}
	}
	return nil
}

func checkAndDeleteVolumes(vols []types.Volume) error {
	fmt.Println("Are you sure you would like to delete the following volumes?")
	for _, v := range vols {
		fmt.Printf("%v ", *v.VolumeId)
	}
	fmt.Println()
	fmt.Println()

	for ok := false; !ok; {
		fmt.Printf("Delete or cancel [d/c]? ")
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		text = strings.TrimSuffix(text, "\n")
		switch text {
		case "c":
			return new(UserCancelError)
		case "d":
			ok = true
		}
	}
	return deleteVolumes(vols)
}
