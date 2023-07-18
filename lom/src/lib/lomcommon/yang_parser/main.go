package main

import (
    "fmt"
    "lom/src/lib/lomcommon/yang_parser/yang_utils"
    "os"
    "strings"
)

const (
    device_health_actions_configs           string = "device-health-actions-configs"
    device_health_actions_configs_file_name string = "device-health-actions-configs.yang"

    device_health_bindings_configs           string = "device-health-bindings-configs"
    device_health_bindings_configs_file_name string = "device-health-bindings-configs.yang"

    device_health_globals_configs           string = "device-health-global-configs"
    device_health_globals_configs_file_name string = "device-health-global-configs.yang"

    device_health_procs_configs           string = "device-health-procs-configs"
    device_health_procs_configs_file_name string = "device-health-procs-configs.yang"

    actions_conf_folder_name string = "actions.confd"
    bindings_conf_file_name  string = "bindings.conf.json"
    globals_conf_file_name   string = "globals.conf.json"
    procs_conf_file_name     string = "procs.conf.json"
)

func main() {

    yang_folder := os.Args[1]
    config_folder := os.Args[2]
    actionsFolderPath := yang_folder + "/actions"
    actionfiles, readErr := os.ReadDir(actionsFolderPath)
    if readErr != nil {
        fmt.Printf("Error reading actionsFolderPath %s. Err %v", actionfiles, readErr)
        return
    }

    for _, file := range actionfiles {
        moduleName := strings.TrimSuffix(file.Name(), ".yang")
        if moduleName != "device-health-actions-common" && moduleName != "device-health-schema-publish-common" {
            actionMapping, err := yang_utils.GetMappingForActionsYangConfig(moduleName, actionsFolderPath+"/"+file.Name())
            if len(err) > 0 {
                fmt.Printf("Error getting mapping for Actions Yang config file %s", moduleName)
                return
            }
            writeError := yang_utils.WriteJsonIntoFile(actionMapping, config_folder+"/"+actions_conf_folder_name, moduleName+".conf.json")
            if writeError != nil {
                fmt.Printf("Writing actions conf failed. %s", moduleName)
                return
            }
        }
    }

    bindingsMapping, err := yang_utils.GetMappingForBindingsYangConfig(device_health_bindings_configs, yang_folder+"/"+device_health_bindings_configs_file_name)
    if len(err) > 0 {
        fmt.Printf("Error getting mapping for Bindings Yang config file")
        return
    }
    writeError := yang_utils.WriteJsonIntoFile(bindingsMapping, config_folder, bindings_conf_file_name)
    if writeError != nil {
        fmt.Printf("Writing bindings conf failed.")
        return
    }

    globalsMapping, err := yang_utils.GetMappingForGlobalsYangConfig(device_health_globals_configs, yang_folder+"/"+device_health_globals_configs_file_name)
    if len(err) > 0 {
        fmt.Printf("Error getting mapping for Globals Yang config file")
        return
    }
    writeError = yang_utils.WriteJsonIntoFile(globalsMapping, config_folder, globals_conf_file_name)
    if writeError != nil {
        fmt.Printf("Writing globals conf failed.")
        return
    }

    procsMapping, err := yang_utils.GetMappingForProcsYangConfig(device_health_procs_configs, yang_folder+"/"+device_health_procs_configs_file_name)
    if len(err) > 0 {
        fmt.Printf("Error getting mapping for Procs Yang config file")
        return
    }
    writeError = yang_utils.WriteJsonIntoFile(procsMapping, config_folder, procs_conf_file_name)
    if writeError != nil {
        fmt.Printf("Writing procs conf failed.")
        return
    }
}
