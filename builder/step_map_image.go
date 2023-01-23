package builder

import (
	"context"
	"fmt"
	"os/exec"
    "regexp"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepMapImage maps system image to /dev/loopX
type StepMapImage struct {
	ResultKey  string
	loopDevice string
}

// Run the step
func (s *StepMapImage) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	image := config.ImageConfig.ImagePath

	// ask losetup to find empty device and map image
	ui.Message(fmt.Sprintf("mapping image %s to free loopback device", image))

	out, err := exec.Command("kpartx", "-avs", image).CombinedOutput()

	if err != nil {
		ui.Error(fmt.Sprintf("error kpartx -avs %v: %s", err, string(out)))
		return multistep.ActionHalt
	}
    
    outstr := string(out)
    loopDeviceName := regexp.MustCompile("loop\\d+").FindString(outstr)
    partitionDeviceNames := regexp.MustCompile("loop\\d+p\\d+").FindAllString(outstr, -1)
    ui.Message(fmt.Sprintf("Using loop device: %s. Partition devices: %v. kpartx output: %s", loopDeviceName, partitionDeviceNames, outstr))
    
    for _, partitionDeviceName := range partitionDeviceNames {
        partitionDevice := "/dev/mapper/" + partitionDeviceName
        symlink := "/dev/" + partitionDeviceName
        ui.Message(fmt.Sprintf("creating symlink %s for partition device %s", symlink, partitionDevice))
        exec.Command("ln", "-s", partitionDevice, symlink)
    }
    s.loopDevice = "/dev/" + loopDeviceName

	state.Put(s.ResultKey, s.loopDevice)
	ui.Message(fmt.Sprintf("image %s mapped to %s", image, s.loopDevice))

	return multistep.ActionContinue
}

// Cleanup after step execution
func (s *StepMapImage) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packer.Ui)

	// Warning: Busy device will prevent detaching loop device from file
	// https://github.com/util-linux/util-linux/issues/484
	out, err := exec.Command("losetup", "--detach", s.loopDevice).CombinedOutput()
	if err != nil {
		ui.Error(fmt.Sprintf("error while unmounting loop device %v: %s", err, string(out)))
	}
}
