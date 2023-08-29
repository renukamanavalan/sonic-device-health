package linkcrc_utils

import (
    "context"
    "errors"
    "fmt"
    "github.com/go-redis/redis"
    "lom/src/plugins/plugins_files/sonic/plugin_integration_tests/utils"
    "strings"
    "time"
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

type InterfaceStatus struct {
    admin_status string
    oper_status  string
}

func MockRedisWithLinkCrcCounters(pattern string, mockTimeInMinutes int, interfaces []string, ctx context.Context) {
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
    for _, v := range interfaces {
        mockedLinks[v] = interfaceToOidMapping[v]
    }

    // Set admin and oper status as up for mocked links.
    var appDbClient = redis.NewClient(&redis.Options{
        Addr:     redis_address,
        Password: redis_password,
        DB:       redis_app_db,
    })

    initialInterfaceStatuses, err := SaveInterfaceStatuses(appDbClient, mockedLinks)
    defer RestoreInterfaceStatuses(appDbClient, initialInterfaceStatuses)
    if err != nil {
        utils.PrintError("Could not save the initial interface statuses locally.")
        return
    }

    ifStatusMock := map[string]interface{}{admin_status: ifUp, oper_status: ifUp}
    for k, _ := range mockedLinks {
        _, err = appDbClient.HMSet("PORT_TABLE:"+k, ifStatusMock).Result()
        if err != nil {
            utils.PrintError("Error mocking admin and oper status for interface %s. Err: %v", k, err)
            return
        } else {
            utils.PrintInfo("Successfully Mocked admin and oper status for interface %s", k)
        }
    }

    var ifInErrors float64
    var ifInUnicastPackets float64
    var ifOutUnicastPackets float64
    var ifOutErrors float64
    ifInErrors = 100
    ifInUnicastPackets = 101
    ifOutUnicastPackets = 1100
    ifOutErrors = 1

    mockTimer := time.NewTimer(time.Duration(mockTimeInMinutes) * time.Minute)
    mockIntervalTicker := time.NewTicker(30 * time.Second)

    /* 0 indicates non outlier, 1 indicates outlier with non-zero counters */
    patternSlice := strings.Split(pattern, ",")
    patternLength := len(patternSlice)
    patternIndex := 0

loop:
    for {
        // Start mocking immediately
        datapoint := map[string]interface{}{"SAI_PORT_STAT_IF_IN_ERRORS": ifInErrors, "SAI_PORT_STAT_IF_IN_UCAST_PKTS": ifInUnicastPackets, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS": ifOutUnicastPackets, "SAI_PORT_STAT_IF_OUT_ERRORS": ifOutErrors}
        for ifName, oidMapping := range mockedLinks {
            _, err := countersDbClient.HMSet(counters_db+oidMapping, datapoint).Result()
            if err != nil {
                utils.PrintError("Error mocking redis data for interface %s. Err %v", ifName, err)
                return
            } else {
                if patternIndex == 0 {
                    utils.PrintInfo("Initial mocking of redis data for interface Succeeded. %s", ifName)
                } else {
                    utils.PrintInfo("Successfuly mocked redis data for interface %s. TimeInUtc: %s. Outlier: %t", ifName, time.Now().UTC().String(), patternSlice[(patternIndex-1)%patternLength] == "1")
                }
            }
        }

        if patternSlice[(patternIndex)%patternLength] == "1" {
            ifInErrors = ifInErrors + 600
            ifInUnicastPackets = ifInUnicastPackets + 1005
            ifOutUnicastPackets = ifOutUnicastPackets + 1005
            ifOutErrors = ifOutErrors + 30
        } else {
            ifInErrors = ifInErrors + 200
            ifInUnicastPackets = ifInUnicastPackets + 300000015
            ifOutUnicastPackets = ifOutUnicastPackets + 1005
            ifOutErrors = ifOutErrors + 30
        }
        patternIndex++

        select {
        case <-mockTimer.C:
            break loop
        case <-mockIntervalTicker.C:
            continue
        case <-ctx.Done():
            break loop
        }
    }
    utils.PrintInfo("Done mocking redis")
}

func SaveInterfaceStatuses(redisClient *redis.Client, mockedLinks map[string]string) (map[string]*InterfaceStatus, error) {
    initialInterfaceStatuses := make(map[string]*InterfaceStatus)
    for ifName, _ := range mockedLinks {
        adminStatus, operStatus, err := getInterfaceStatus(redisClient, ifName)

        if err != nil {
            /* Send partial result */
            return initialInterfaceStatuses, err
        }

        intStatus := InterfaceStatus{admin_status: adminStatus, oper_status: operStatus}
        initialInterfaceStatuses[ifName] = &intStatus
    }

    return initialInterfaceStatuses, nil
}

func RestoreInterfaceStatuses(redisClient *redis.Client, initialStatuses map[string]*InterfaceStatus) error {
    for ifName, intStatus := range initialStatuses {
        ifStatusMock := map[string]interface{}{admin_status: intStatus.admin_status, oper_status: intStatus.oper_status}
        _, err := redisClient.HMSet("PORT_TABLE:"+ifName, ifStatusMock).Result()
        if err != nil {
            utils.PrintError("Error restoring admin and oper status for interface %s. Err: %v", ifName, err)
        } else {
            utils.PrintInfo("Successfully restored admin and oper status for interface %s", ifName)
        }
    }

    utils.PrintInfo("Done restoring statuses of all interfaces.")
    return nil
}

func getInterfaceStatus(redisClient *redis.Client, interfaName string) (string, string, error) {
    interfaceStatusKey := "PORT_TABLE:" + interfaName
    fields := []string{"admin_status", "oper_status"}
    result, err := redisClient.HMGet(interfaceStatusKey, fields...).Result()
    if err != nil {
        return "", "", errors.New(fmt.Sprintf("getInterfaceStatus.HmGet Failed for key (%s). err: (%v)", interfaceStatusKey, err))
    }
    return result[0].(string), result[1].(string), nil
}
