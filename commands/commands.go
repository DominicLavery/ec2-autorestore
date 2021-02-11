package commands

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
)

var ec2c *ec2.EC2
//const rootVol = "/dev/xvda"
const rootVol = "/dev/sda1" //TODO make configurable with flag

func InitialiseCommands(e *ec2.EC2) {
	ec2c = e
}

func shutdownInstances(instancesMap map[string]*ec2.Instance) []*string {
	var ids []*string
	log.Println("Attempting to shutdown instances")
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
	log.Printf("Attempting to restart")
	_, err := ec2c.StartInstances(new(ec2.StartInstancesInput).SetInstanceIds(ids))
	if err != nil {
		log.Println("Can't start instance", err)
	}
}