package commands

import (
	"github.com/spf13/cobra"
	"log"
)

func PruneCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "prune",
		Short: "Cleans up old EC2 resources",
		Long:  `TODO`,
	}
	command.AddCommand(&cobra.Command{
		Use:   "volumes",
		Short: "Prunes unused backup volumes",
		Run: func(cmd *cobra.Command, args []string) {
			pruneVolumes(args[0])
		},
	})

	command.AddCommand(&cobra.Command{
		Use:   "snapshots [backup id]",
		Short: "Prunes snapshots with the given ID",
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
