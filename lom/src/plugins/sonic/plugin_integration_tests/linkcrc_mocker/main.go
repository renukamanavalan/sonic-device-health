package main

import (
    "os"
    "os/exec"
    "lom/src/plugins/sonic/plugin_integration_tests/linkcrc_mocker/linkcrc_utils"
    "lom/src/plugins/sonic/plugin_integration_tests/utils"
)

const (
counter_poll_disable_command = "sudo counterpoll port disable"
)

func main() {
    // Pre - setup
    utils.PrintInfo("Starting Link CRC Detection plugin integration test.")
    _, err := exec.Command("/bin/sh", "-c", counter_poll_disable_command).Output()
    if err != nil {
        utils.PrintError("Error disabling counterpoll on switch %v", err)
    } else {
        utils.PrintInfo("Successfuly Disabled counterpoll")
    }

	testId := os.Args[1]
	interfaces := os.Args[2:]

	switch testId {
	case "0":
		linkcrc_utils.MockRedisWithLinkCrcCounters("1", 10, interfaces)
		break
	case "1":
		linkcrc_utils.MockRedisWithLinkCrcCounters("0", 10, interfaces)
		break
	case "2":
		linkcrc_utils.MockRedisWithLinkCrcCounters("1,0,0,0", 10, interfaces)
		break
	case "3":
		linkcrc_utils.MockRedisWithLinkCrcCounters("1,0,0,0,0", 2, interfaces)
		break
	default:
		utils.PrintError("Invalid test Id %d", testId)
	}

}

