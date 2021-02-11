package commands

import "github.com/aws/aws-sdk-go/service/ec2"

var ec2c *ec2.EC2
const rootVol = "/dev/xvda"
//"/dev/sda1"

func InitialiseCommands(e *ec2.EC2) {
	ec2c = e
}
