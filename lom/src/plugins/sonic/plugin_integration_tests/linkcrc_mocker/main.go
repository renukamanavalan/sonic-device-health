package main

import (
     "github.com/go-redis/redis"
     "time"
     "lom/src/plugins/sonic/plugin_integration_tests/utils"
     "os"
     "os/exec"
     "strconv"
)

const (
    redis_address                    = "localhost:6379"
    redis_counters_db                = 2
    redis_app_db                     = 0
    fileName                         = "./LINK_CRC_COUNTERS"
    redis_password                   = ""
    counter_poll_disable_command     = "sudo counterpoll port disable"
    action_name                      = "link_crc_detection"
    detection_type                   = "detection"
    counter_poll_enable_command      = "sudo counterpoll port enable"
    counters_port_name_map_redis_key = "COUNTERS_PORT_NAME_MAP"
    admin_status                     = "admin_status"
    oper_status                      = "oper_status"
    ifUp                             = "up"
    counters_db                      = "COUNTERS:"
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
    MockRedisWithLinkCrcCounters()
}

func MockRedisWithLinkCrcCounters() {
        var countersDbClient = redis.NewClient(&redis.Options{
		Addr:     redis_address,
		Password: redis_password,
		DB:       redis_counters_db,
	})

	// Get all ifName to oid mappings.
	interfaceToOidMapping, err := countersDbClient.HGetAll(counters_port_name_map_redis_key).Result()
	if err != nil {
		utils.PrintError("Error fetching counters port name map %v", err)
		return
	}

	// Filter only those required as per the links passed through arguments
	mockedLinks := make(map[string]string)
	for _, v := range os.Args[3:] {
		mockedLinks[v] = interfaceToOidMapping[v]
	}

        var ifInErrors float64
	var ifInUnicastPackets float64
	var ifOutUnicastPackets float64
	var ifOutErrors float64
	ifInErrors = 100
	ifInUnicastPackets = 101
	ifOutUnicastPackets = 1100
	ifOutErrors = 1

	period, err := strconv.Atoi(os.Args[2])
	timer1 := time.NewTimer(time.Duration(period) * time.Minute)

	simulate, err := strconv.Atoi(os.Args[1]) /* 0 indicates simulation, 1 indicates non-simulation with non-zero counters */

loop:
	for {
		select {
		case <-timer1.C:
			break loop
		default:

			datapoint := map[string]interface{}{"SAI_PORT_STAT_IF_IN_ERRORS": ifInErrors, "SAI_PORT_STAT_IF_IN_UCAST_PKTS": ifInUnicastPackets, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS": ifOutUnicastPackets, "SAI_PORT_STAT_IF_OUT_ERRORS": ifOutErrors}
			for ifName, oidMapping := range mockedLinks {
				_, err := countersDbClient.HMSet(counters_db+oidMapping, datapoint).Result()
				if err != nil {
					utils.PrintError("Error mocking redis data for interface %s. Err %v", ifName, err)
					return
				} else {
					utils.PrintInfo("Successfuly mocked redis data for interface %s", ifName)
				}
			}
			time.Sleep(2 * time.Second)

			if simulate == 0 {
				ifInErrors = ifInErrors + 40
				ifInUnicastPackets = ifInUnicastPackets + 67
				ifOutUnicastPackets = ifOutUnicastPackets + 67
				ifOutErrors = ifOutErrors + 2
			} else {
				ifInErrors = ifInErrors + 14
				ifInUnicastPackets = ifInUnicastPackets + 20000001
				ifOutUnicastPackets = ifOutUnicastPackets + 67
				ifOutErrors = ifOutErrors + 2
			}

		}
	}
}
