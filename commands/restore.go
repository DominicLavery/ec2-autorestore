package commands

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
	"log"
)

func RestoreCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restore [backupID]",
		Short: "Restores an instance from a backup",
		Long:  `Restores an instance from a backup to snapshots with the given backup id,`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			restore(args[0])
		},
	}
}

func restore(backupId string) {
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
	instancesMap, snapshotMap := gatherInfo(snapshots)
	newVolumesMap := make(map[string]*ec2.Volume)

	//Make volumes
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

	ids := shutdownInstances(instancesMap)

	//detach the old volumes
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

	err := ec2c.WaitUntilVolumeAvailable(new(ec2.DescribeVolumesInput).SetVolumeIds(detachedVolumes))
	if err != nil {
		//todo reattach removed ones
		log.Fatal("Can't remove vol", err)
	}

	//Attach the new volumes
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

	restartInstances(ids)
	//TODO Add an option to delete the detached volumes? And the snapshots?
}

func gatherInfo(snapshots []*ec2.Snapshot) (map[string]*ec2.Instance, map[string]*ec2.Snapshot){
	instancesMap := make(map[string]*ec2.Instance)
	snapshotMap := make(map[string]*ec2.Snapshot)
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
	return instancesMap, snapshotMap
}