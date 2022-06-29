package commands

import (
	"log"

	"github.com/spf13/cobra"
)

func PruneCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "prune",
		Short: "Cleans up old EC2 resources",
		Long:  `Cleans up old EC2 resources left over from other commands based on backup IDs`,
	}
	command.AddCommand(&cobra.Command{
		Use:   "volumes [backup id]",
		Short: "Prunes unused volumes",
		Long:  "Prunes unused volumes left behind after a restore that have the given backup id",
		Run: func(cmd *cobra.Command, args []string) {
			pruneVolumes(args[0])
		},
	})

	command.AddCommand(&cobra.Command{
		Use:   "snapshots [backup id]",
		Short: "Prunes snapshots",
		Long:  "Prunes snapshots that have the given backup id",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pruneSnapshots(args[0])
		},
	})

	return command
}

func pruneSnapshots(backupId string) {
	snapshots, err := getSnapshots(backupId)
	if err != nil {
		log.Fatalf("Could not find snapshots because: %v", err)
	} else if len(snapshots) < 1 {
		log.Fatalf("No snapshots with backup id \"%v\"", backupId)
	}

	err = checkAndDeleteSnaps(snapshots)
	if err != nil {
		log.Fatalf("Unable to delete snapshot because %v", err)
	}
}

func pruneVolumes(backupId string) {
	volumes, err := getVolumes(backupId)
	if err != nil {
		log.Fatalf("Could not find volumes because: %v", err)
	} else if len(volumes) < 1 {
		log.Fatalf("No volumes with backup id \"%v\"", backupId)
	}

	err = checkAndDeleteVolumes(volumes)
	if err != nil {
		log.Fatalf("Unable to delete snapshot because %v", err)
	}
}
