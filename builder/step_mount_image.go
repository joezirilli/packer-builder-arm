package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
    "strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	cfg "github.com/mkaczanowski/packer-builder-arm/config"
)

func sortMountablePartitions(partitions []cfg.Partition, reverse bool) []cfg.Partition {
	mountable := []cfg.Partition{}

	for i, partition := range partitions {
		partition.Index = i + 1
		if partition.Mountpoint != "" {
			mountable = append(mountable, partition)
		}
	}

	sort.Slice(mountable, func(i, j int) bool {
		if reverse {
			return mountable[i].Mountpoint > mountable[j].Mountpoint
		}
		return mountable[i].Mountpoint < mountable[j].Mountpoint
	})

	return mountable
}

// StepMountImage mounts partition to selected mountpoints
type StepMountImage struct {
	FromKey     string
	ResultKey   string
	MountPath   string
	mountpoints []string
}

// Run the step
func (s *StepMountImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	loopDevice := state.Get(s.FromKey).(string)

	if len(s.MountPath) > 0 {
		err := os.MkdirAll(s.MountPath, os.ModePerm)
		if err != nil {
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	} else {
		tempdir, err := os.MkdirTemp("", "")
		if err != nil {
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		s.MountPath = tempdir
	}

	partitions := sortMountablePartitions(config.ImageConfig.ImagePartitions, false)
	for _, partition := range partitions {
		mountpoint := filepath.Join(s.MountPath, partition.Mountpoint)
        loopDeviceActualPath = strings.Replace(loopDevice, "/dev/", "/dev/mapper/", -1)
		device := fmt.Sprintf("%sp%d", loopDeviceActualPath, partition.Index)

		if err := os.MkdirAll(mountpoint, 0755); err != nil {
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		ui.Message(fmt.Sprintf("mounting %s to %s", device, mountpoint))
		mountout, err := exec.Command("mount", "-o", "discard", device, mountpoint).CombinedOutput()
		if err != nil {
            ui.Error(fmt.Sprintf("failed to mount device %s to %s: %s. out: %s", device, mountpoint, err.Error(), string(mountout)))
			return multistep.ActionHalt
		}

		s.mountpoints = append(s.mountpoints, mountpoint)
	}

	state.Put(s.ResultKey, s.MountPath)

	return multistep.ActionContinue
}

// Cleanup after step execution
func (s *StepMountImage) Cleanup(state multistep.StateBag) {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	if s.MountPath != "" {
		partitions := sortMountablePartitions(config.ImageConfig.ImagePartitions, true)
		for _, partition := range partitions {
			mountpoint := filepath.Join(s.MountPath, partition.Mountpoint)
			ui.Message(fmt.Sprintf("unmounting %s", mountpoint))
			umountout, err := exec.Command("umount", mountpoint).CombinedOutput()
			if err != nil {
                ui.Error(fmt.Sprintf("failed to unmount %s: %s. out: %s", mountpoint, err.Error(), string(umountout)))
			}
		}
		s.mountpoints = nil

		if err := os.Remove(s.MountPath); err != nil {
			ui.Error(fmt.Sprintf("failed to remove %s: %s", s.MountPath, err.Error()))
		}

		s.MountPath = ""
	}
}
