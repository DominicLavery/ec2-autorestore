package commands

import (
	"bufio"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
	"os"
	"strings"
)

var ec2c *ec2.EC2

func InitialiseCommands(e *ec2.EC2) {
	ec2c = e
}

func shutdownInstances(instancesMap map[string]*ec2.Instance) []*string {
	//TODO handle situation where instances are shutdown already?
	var ids []*string
	log.Println("Shutting down instances")
	for _, i := range instancesMap {
		ids = append(ids, i.InstanceId)
	}

	input := new(ec2.StopInstancesInput).SetInstanceIds(ids)
	_, err := ec2c.StopInstances(input)
	if err != nil {
		log.Fatal("Can't stop instance", err)
	}
	err = ec2c.WaitUntilInstanceStopped(new(ec2.DescribeInstancesInput).SetInstanceIds(ids))
	if err != nil {
		log.Fatal("Can't stop instance", err)
	}
	return ids
}

func restartInstances(ids []*string) {
	log.Printf("Restartting instances")
	_, err := ec2c.StartInstances(new(ec2.StartInstancesInput).SetInstanceIds(ids))
	if err != nil {
		log.Println("Can't start instance", err)
	}
}

func getSnapshots(backupId string) ([]*ec2.Snapshot, error) {
	var snapshots []*ec2.Snapshot
	filters := []*ec2.Filter{new(ec2.Filter).SetName("tag:autorestore-backupId").SetValues([]*string{aws.String(backupId)})}
	var ownerIds = []*string{aws.String("self")}
	input := new(ec2.DescribeSnapshotsInput).SetOwnerIds(ownerIds).SetFilters(filters)
	for ok := true; ok; {
		out, err := ec2c.DescribeSnapshots(input)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, out.Snapshots...)
		if ok = out.NextToken != nil; ok {
			input.SetNextToken(*out.NextToken)
		}
	}
	return snapshots, nil
}

func checkAndDeleteSnaps(snaps []*ec2.Snapshot) error {
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

func deleteSnapshots(snaps []*ec2.Snapshot) error {
	for _, s := range snaps {
		log.Printf("Deleting snapshot \"%v\"", *s.SnapshotId)
		_, err := ec2c.DeleteSnapshot(new(ec2.DeleteSnapshotInput).SetSnapshotId(*s.SnapshotId))
		if err != nil {
			return err
		}
	}
	return nil
}


func getVolumes(backupId string) ([]*ec2.Volume, error) {
	var vols []*ec2.Volume
	filters := []*ec2.Filter{
		new(ec2.Filter).SetName("tag:autorestore-backupId").SetValues([]*string{aws.String(backupId)}),
		new(ec2.Filter).SetName("status").SetValues([]*string{aws.String("available")}),
	}
	input := new(ec2.DescribeVolumesInput).SetFilters(filters)
	for ok := true; ok; {
		out, err := ec2c.DescribeVolumes(input)
		if err != nil {
			return nil, err
		}
		vols = append(vols, out.Volumes...)
		if ok = out.NextToken != nil; ok {
			input.SetNextToken(*out.NextToken)
		}
	}
	return vols, nil
}

func deleteVolumes(vols []*ec2.Volume) error {
	for _, v := range vols {
		log.Printf("Deleting snapshot \"%v\"", *v.VolumeId)
		_, err := ec2c.DeleteVolume(new(ec2.DeleteVolumeInput).SetVolumeId(*v.VolumeId))
		if err != nil {
			return err
		}
	}
	return nil
}

func checkAndDeleteVolumes(vols []*ec2.Volume) error {
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