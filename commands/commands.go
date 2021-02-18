package commands

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
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