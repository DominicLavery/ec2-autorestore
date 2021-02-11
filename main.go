package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"log"
)

var backup = false
var backupId = "Doms-test-1"

const rootVol = "/dev/xvda"

var ec2c *ec2.EC2

//"/dev/sda1"
func main() {
	sess := session.Must(session.NewSession(new(aws.Config).WithRegion("eu-west-2")))
	creds := stscreds.NewCredentials(sess, "arn:aws:iam::347316231722:role/fromCore")
	ec2c = ec2.New(sess, &aws.Config{Credentials: creds})

	if backup {
		createSnapshots()
	} else {
		restore()
	}
}

func createSnapshots() {
	filters := []*ec2.Filter{new(ec2.Filter).SetName("tag:backup").SetValues([]*string{aws.String("test")})}
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

func restore() {
	filters := []*ec2.Filter{new(ec2.Filter).SetName("tag:autorestore-backupId").SetValues([]*string{aws.String(backupId)})}
	var strings = []*string{aws.String("347316231722")}
	input := new(ec2.DescribeSnapshotsInput).SetOwnerIds(strings).SetFilters(filters)

	for ok := true; ok; {
		out, err := ec2c.DescribeSnapshots(input)
		if err != nil {
			log.Fatal("Can't get snapshots", err)
		}
		processSnapshots(out.Snapshots)
		if ok = out.NextToken != nil; ok {
			input.SetNextToken(*out.NextToken)
		}
	}
}

func processSnapshots(snapshots []*ec2.Snapshot) {

	instancesMap := make(map[string]*ec2.Instance)
	snapshotMap := make(map[string]*ec2.Snapshot)
	newVolumesMap := make(map[string]*ec2.Volume)

	// Make sure the instance exists before we create vols
	for _, s := range snapshots {
		var tagVal string
		for _, tag := range s.Tags {
			if *tag.Key == "autorestore-instanceId" {
				tagVal = *tag.Value
				break
			}
		}
		if tagVal == "" {
			log.Fatalf("Snapshot %s did not have a tag of autorestore-instanceId\n", *s.SnapshotId)
		}
		log.Printf("Node %s found", tagVal)
		i, err := ec2c.DescribeInstances(new(ec2.DescribeInstancesInput).SetInstanceIds([]*string{aws.String(tagVal)}))
		if err != nil {
			log.Fatalf("Couldn't get instance because %s", err)
		}
		instancesMap[tagVal] = i.Reservations[0].Instances[0]
		snapshotMap[tagVal] = s
	}

	for id, i := range instancesMap {
		log.Printf("Attempting to create volume for %s", id)
		v, err := ec2c.CreateVolume(new(ec2.CreateVolumeInput).
			SetSnapshotId(*snapshotMap[id].SnapshotId).
			SetAvailabilityZone(*i.Placement.AvailabilityZone))
		if err != nil {
			//TODO delete vols
			log.Fatalf("Couldn't create volume because %s", err)
		}
		newVolumesMap[id] = v
	}

	var ids []*string

	for id, i := range instancesMap {
		log.Printf("Attempting to shutdown %s", id)
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

	var detachedVolumes []*string
	for id, i := range instancesMap {
		for _, b := range i.BlockDeviceMappings {
			if *b.DeviceName == rootVol {
				log.Printf("Attempting to detach vol for %s from %s\n", *b.Ebs.VolumeId, id)
				_, err := ec2c.DetachVolume(new(ec2.DetachVolumeInput).SetVolumeId(*b.Ebs.VolumeId))
				if err != nil {
					//todo reattach removed ones
					log.Fatal("Can't remove vol", err)
				}
				detachedVolumes = append(detachedVolumes, b.Ebs.VolumeId)
			}
		}
	}

	err = ec2c.WaitUntilVolumeAvailable(new(ec2.DescribeVolumesInput).SetVolumeIds(detachedVolumes))
	if err != nil {
		//todo reattach removed ones
		log.Fatal("Can't remove vol", err)
	}

	var attachedVolumes []*string
	for id, v := range newVolumesMap {
		log.Printf("Attempt to attach vol %s to %s", *v.VolumeId, id)
		_, err := ec2c.AttachVolume(new(ec2.AttachVolumeInput).SetInstanceId(id).SetVolumeId(*v.VolumeId).SetDevice(rootVol))
		if err != nil {
			//TODO Some better handling
			log.Println("Can't attach", err)
		}
		attachedVolumes = append(attachedVolumes, v.VolumeId)
	}

	err = ec2c.WaitUntilVolumeInUse(new(ec2.DescribeVolumesInput).SetVolumeIds(attachedVolumes))
	if err != nil {
		log.Fatal("Can't reattach vol", err)
	}

	log.Printf("Attempting to restart")
	_, err = ec2c.StartInstances(new(ec2.StartInstancesInput).SetInstanceIds(ids))
	if err != nil {
		log.Println("Can't start instance", err)
	}
}
