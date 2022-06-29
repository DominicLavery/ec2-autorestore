package commands

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/cobra"
)

func RestoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore [backupID]",
		Short: "Restores an instance from a backup",
		Long:  `Restores an instance from a backup to snapshots with the given backup id,`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			restore(args[0])
		},
	}
	cmd.Flags().BoolVarP(&rmVolumes, "dvols", "d", false, "Deletes the volumes detached from the instance")
	return cmd
}

func restore(backupId string) {
	snapshots, err := getSnapshots(backupId)
	if err != nil {
		log.Fatal("Can't get snapshots", err)
	}
	if len(snapshots) == 0 {
		log.Fatalf("there are no snapshots with backup id of %v for this account", backupId)
	}
	processSnapshots(snapshots)
}

func processSnapshots(snapshots []types.Snapshot) {
	instancesMap, snapshotMap := gatherInfo(snapshots)
	newVolumesMap := make(map[string]string)

	//Make volumes
	for id, i := range instancesMap {
		ts := []types.TagSpecification{{Tags: snapshotMap[id].Tags, ResourceType: "volume"}}
		log.Printf("Attempting to create volume for %s", id)
		v, err := ec2c.CreateVolume(context.TODO(), &ec2.CreateVolumeInput{
			SnapshotId:        snapshotMap[id].SnapshotId,
			AvailabilityZone:  i.Placement.AvailabilityZone,
			TagSpecifications: ts,
		})

		if err != nil {
			//TODO delete vols
			log.Fatalf("Couldn't create volume because %s", err)
		}
		newVolumesMap[id] = *v.VolumeId
	}

	ids := shutdownInstances(instancesMap)

	//detach the old volumes
	var detachedVolIds []string
	for id, i := range instancesMap {
		for _, b := range i.BlockDeviceMappings {
			if *b.DeviceName == *i.RootDeviceName {
				log.Printf("Attempting to detach vol for %s from %s\n", *b.Ebs.VolumeId, id)
				_, err := ec2c.DetachVolume(context.TODO(), &ec2.DetachVolumeInput{VolumeId: b.Ebs.VolumeId})
				if err != nil {
					//todo reattach removed ones
					log.Fatal("Can't remove vol", err)
				}
				detachedVolIds = append(detachedVolIds, *b.Ebs.VolumeId)
			}
		}
	}
	volAvail := ec2.NewVolumeAvailableWaiter(ec2c)
	maxWaitTime := 5 * time.Minute
	params := &ec2.DescribeVolumesInput{VolumeIds: detachedVolIds}
	err := volAvail.Wait(context.TODO(), params, maxWaitTime)

	if err != nil {
		//todo reattach removed ones
		log.Fatal("Can't remove vol", err)
	}

	//Attach the new volumes
	var attachedVolumes []string
	for id, v := range newVolumesMap {
		log.Printf("Attaching vol %s to %s", v, id)
		i := instancesMap[id]
		_, err := ec2c.AttachVolume(context.TODO(), &ec2.AttachVolumeInput{
			InstanceId: aws.String(id),
			VolumeId:   aws.String(v),
			Device:     i.RootDeviceName,
		})
		if err != nil {
			//TODO Some better handling
			log.Printf("Failed to attach %v because: %v\n", v, err)
		}
		attachedVolumes = append(attachedVolumes, v)
	}

	volInUse := ec2.NewVolumeInUseWaiter(ec2c)
	params = &ec2.DescribeVolumesInput{VolumeIds: attachedVolumes}
	err = volInUse.Wait(context.TODO(), params, maxWaitTime)

	if err != nil {
		log.Fatal("Can't reattach vol", err)
	}

	restartInstances(ids)
	if rmVolumes {
		detachedVols, err := getVolumesByIds(detachedVolIds)
		if err != nil {
			log.Fatal("Unable to fetch information to clean up volumes: ", err)
		}
		err = checkAndDeleteVolumes(detachedVols)
		if err != nil {
			log.Fatal("Can't delete volumes because ", err)
		}
	}
}

func gatherInfo(snapshots []types.Snapshot) (map[string]types.Instance, map[string]types.Snapshot) {
	instancesMap := make(map[string]types.Instance)
	snapshotMap := make(map[string]types.Snapshot)
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
		i, err := ec2c.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{InstanceIds: []string{tagVal}})
		if err != nil {
			log.Fatalf("Couldn't get instance because %s", err)
		}
		instancesMap[tagVal] = i.Reservations[0].Instances[0]
		snapshotMap[tagVal] = s
	}
	return instancesMap, snapshotMap
}
