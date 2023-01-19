package builder

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
    "time"

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

	out, err := exec.Command("losetup", "--find", "--partscan", "--show", "--verbose", image).CombinedOutput()
    ui.Message("Waiting 10 seconds")
    time.Sleep(10 * time.Second)
    outlsdev, errlsdev := exec.Command("ls", "/dev").CombinedOutput()
    if errlsdev != nil {
        ui.Error(fmt.Sprintf("error ls /dev %v: %s", errlsdev, string(outlsdev)))
    } else {
        ui.Message(fmt.Sprintf("ls /dev result: \n%s", string(outlsdev)))
    }
    
    outlosetupall, errlosetupall := exec.Command("losetup", "--all").CombinedOutput()
    if errlosetupall != nil {
        ui.Error(fmt.Sprintf("error losetup --all %v: %s", errlosetupall, string(outlosetupall)))
    } else {
        ui.Message(fmt.Sprintf("losetup --all result: %s", string(outlosetupall)))
    }

	if err != nil {
		ui.Error(fmt.Sprintf("error losetup --find --partscan %v: %s", err, string(out)))
		return multistep.ActionHalt
    } else {
        ui.Message(fmt.Sprintf("losetup output: %s", string(out)))
    }
	s.loopDevice = strings.Trim(string(out), "\n")

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
