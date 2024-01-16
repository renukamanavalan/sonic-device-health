from __future__ import print_function  # Python 2/3 compatibility
import os
import subprocess
import signal
import time
import json
import sys
import arista_eapi_helper as eapi_helper
import arista_cli_helper as cli_helper

from common import *

PROC_CONF_FILE = 'procs.conf.json'
ACTIONS_CONF_FILE = 'actions.conf.json'
BINDINGS_CONF_FILE = 'bindings.conf.json'
GLOBALS_CONF_FILE = 'globals.conf.json'

# Function to get proc keys
def get_procs_keys(script_dir):
    try:
        # Read the contents of procs.conf.json
        with open(os.path.join(script_dir, '..', 'config', PROC_CONF_FILE), 'r') as f:
            data = json.load(f)
        # Get the keys under the "procs" object
        procs_keys = list(data.get('procs', {}).keys())  # Convert to list for Python 2.7
        # Check if the "procs" object is empty
        if not procs_keys:
            print("Error: The 'procs' object is empty in procs.conf.json.")
            sys.exit(1)
        #print("procs.conf.json keys: " + str(procs_keys))
        return procs_keys
    except IOError as e:
        print("Error: procs.conf.json not found in the specified path. {}".format(e))
        sys.exit(1)
    except ValueError as e:
        print("Error: Failed to decode procs.conf.json.{}".format(e))
        sys.exit(1)
    except Exception as e:
        print("Error: An unexpected error occurred while reading procs.conf.json. {}".format(e))
        sys.exit(1)





# Check if at least one argument was provided
if len(sys.argv) < 2:
    print("Error: Please provide at least one installation argument.")
    sys.exit(1)

# Get the first argument passed to the script
arg1 = sys.argv[1]

# Print the value of the argument
print("Installation option : ", arg1)

#signal.signal(signal.SIGINT, signal_handler)

# Get the directory where this script is located
script_dir = os.path.dirname(os.path.abspath(__file__))
print("Script directory: " + str(script_dir) + "\n")

config_files = [PROC_CONF_FILE, ACTIONS_CONF_FILE, BINDINGS_CONF_FILE, GLOBALS_CONF_FILE]

# Check if each configuration file exists
config_file_path = None
for config_file in config_files:
    config_file_path = os.path.join(script_dir, '..', 'config', config_file)
    if not os.path.exists(config_file_path):
        print("Error: config file {} not found.".format(config_file))
        sys.exit(1)

# Check if LoMEngine binary exists
lom_engine_path = os.path.join(script_dir, 'LoMEngine')
if not os.path.exists(lom_engine_path):
    print("Error: LoMEngine binary not found.")
    sys.exit(1)

# Check if LoMPluginMgr binary exists
lom_plugin_mgr_path = os.path.join(script_dir, 'LoMPluginMgr')
if not os.path.exists(lom_plugin_mgr_path):
    print("Error: LoMPluginMgr binary not found.")
    sys.exit(1)

# Check if the LoMCli binary exists
lom_cli_path = os.path.join(script_dir, 'LoMCli')
if not os.path.exists(lom_cli_path):
    print("Error: LoMCli binary not found.")
    sys.exit(1)

# Construct the path to the 'config' directory based on the script's location
config_dir = os.path.join(script_dir, '..', 'config')

# Create an instance of AristaSwitchCLIHelper to enable commands via CLI first
switch_cli = cli_helper.AristaSwitchCLIHelper()
eapi_already_enabled = False

try:
    # Enable Unix eAPI protocol
    success, eapi_already_enabled, result = switch_cli.check_and_enable_unix_eAPI_protocol()
    if success:
        print(result)  # Valid message
    else:
        print("Error: Failed to enable Unix eAPI protocol. {}".format(result))        
        sys.exit(1)
except Exception as e:
    print("An error occurred in the AristaSwitchCLIHelper: {}".format(e))
    sys.exit(1)


# Set environment variables
#os.environ["LOM_CONF_LOCATION"] = os.path.abspath(config_dir)
#os.environ["LOM_RUN_MODE"] = "PROD"
    
# Print the environment variables to verify
#print("LOM_CONF_LOCATION={}, LOM_RUN_MODE={}".format(os.environ['LOM_CONF_LOCATION'], os.environ['LOM_RUN_MODE']))

# Get the list of proc IDs from PROC_CONF_FILE
procs_keys = get_procs_keys(script_dir)
print("procs_keys::::::::::: ", procs_keys, "\n")

# Create an instance of AristaSwitchEAPIHelper to execute commands via eAPI
switch_eapi = eapi_helper.AristaSwitchEAPIHelper()
terminatr_already_enabled = False

try:
    switch_eapi.connect()

    # Extract all daemon information untill now
    daemons_info, error = switch_eapi.extract_daemons_info()
    if error:
        print("Error while extracting daemon info:", error)
        sys.exit(1)

    print("daemons info at startup: ", daemons_info)
    print_with_separator(json.dumps(daemons_info, indent=4))

    # Check and enable TerminAttr daemon
    running, terminatr_already_enabled, error = switch_eapi.check_and_enable_terminattr_daemon()
    if error:
        print("Error: Failed to check and enable TerminAttr. {}".format(error))
        sys.exit(1)
    
    print("TerminAttr daemon is currently: {}".format("Running" if running else "Not running"))
    print("TerminAttr daemon was already enabled: {}".format(terminatr_already_enabled))

    # Get information about all lom-plmgr daemon instances
    lom_plmgr_info, error = switch_eapi.get_daemon_lom_plmgr_info()
    if error:
        print("Error while getting lom-plmgr info:", error)
        sys.exit(1)
    else:
        if not lom_plmgr_info:
            print("No lom-plmgr daemons config is enabled")
        else:
            print("lom-plmgr daemons config exists")

            # Disable all lom-plmgr daemon instances
            for instance_name, instance_info in lom_plmgr_info.items():
                if instance_info.get("Running", False):
                    print("{} is running".format(instance_name))
                else:
                    print("{} is not running.".format(instance_name))

                print("Disabling {}...".format(instance_name))
                    
                # Disable the daemon
                result, error = switch_eapi.disable_daemon(instance_name)
                if error:
                    print("Error while disabling {}: {}".format(instance_name, error))
                    sys.exit(1)
                else:
                    print("{} disabled successfully".format(instance_name))

                # validate if still lom-plmgr config is enabled or not 
                daemons_info, error = switch_eapi.extract_daemons_info()
                if error:
                    print("Error while extracting daemon info:", error)
                    sys.exit(1)

                print("daemons info: ")
                print_with_separator(json.dumps(daemons_info, indent=4))

                print("Validating if {} is still daemon config exists or not".format(instance_name))
                running, error = switch_eapi.is_daemon_config_exists(instance_name)
                if error:
                    print("Error while checking {} status:".format(instance_name), error)
                    sys.exit(1)
                else:
                    if running:
                        print("{} config still exists. Exiting".format(instance_name))
                        sys.exit(1)
                    else:
                        print("{} config is cleaned successfully".format(instance_name))


    # Get information about 'lom-engine' daemon
    lom_engine_info, error = switch_eapi.get_daemon_lom_engine_info()
    if error:
        print("Error while getting lom-engine info:", error)
        sys.exit(1)
    else:
        if not lom_engine_info:
            print("No lom-engine cdaemon config is enabled")
        else:
            print("lom-engine daemon config exists")

            if lom_engine_info.get("Running", False):
                print("lom-engine is running.")
            else:
                print("lom-engine is not running.")
                
            print("Disabling lom-engine...")
                
            # Disable the lom-engine daemon
            result, error = switch_eapi.disable_daemon('lom-engine')
            if error:
                print("Error while disabling lom-engine: {}".format(error))
                sys.exit(1)
            else:
                print("lom-engine disabled successfully")

            print("daemons info: ")
            print_with_separator(json.dumps(daemons_info, indent=4))

            # validate if still lom-engine config is enabled or not 
            print("Validating if lom-engine is still enabled or not")
            daemons_info, error = switch_eapi.extract_daemons_info()
            if error:
                print("Error while extracting daemon info:", error)
                sys.exit(1)

            running, error = switch_eapi.is_daemon_config_exists('lom-engine')
            if error:
                print("Error while checking lom-engine daemon status:", error)
                sys.exit(1)
            else:
                if running:
                    print("lom-engine config still enabled")
                    sys.exit(1)
                else:
                    print("lom-engine config is disabled successfully")


    # start the lom-engine daemon 
    print("\n" + "Starting lom-engine daemon ...")
    start_command = [
        'configure',
        'daemon lom-engine',
        #'exec /mnt/flash/goutham/installation/install/LoMEngine -path=/mnt/flash/goutham/installation/config -mode=PROD',
        'exec {} -path={} -mode=PROD'.format(lom_engine_path, config_dir),
        'no shutdown',
        'exit',
    ]
    print ("start_command: ", start_command)
    start_response, error = switch_eapi.execute_command(start_command)
    if error:
        print("Error while starting lom-engine:", error)
        sys.exit(1)
    else:
        print("lom-engine started successfully" + "\n")

    # validate if lom-engine is running or not
    daemons_info, error = switch_eapi.extract_daemons_info()
    if error:
            print("Error while extracting daemon info:", error)
            sys.exit(1)

    print("daemons info: ")
    print_with_separator(json.dumps(daemons_info, indent=4))
    
    print("Validating if lom-engine is running or not")
    running, error = switch_eapi.is_daemon_running('lom-engine')
    if error:
        print("Error while checking lom-engine status:", error)
        sys.exit(1)
    else:
        if not running:
            print("lom-engine is not running")
            sys.exit(1)
        else:
            print("lom-engine is running successfully" + "\n")

    # Iterate over each proc_id in procs_keys and start lom-plmgr for each proc_id            
    #procs_keys = ["proc-0", "proc-1", "proc-2"]
    for proc_id in procs_keys:
        print("\n" + "Starting lom-plmgr for proc_id {} ...".format(proc_id))
        start_command = [
            'configure',
            'daemon lom-plmgr-{}'.format(proc_id),
            #'exec /mnt/flash/goutham/installation/install/LoMPluginMgr -proc_id={} -syslog_level=7 -path=/mnt/flash/goutham/installation/config -mode=PROD'.format(proc_id),
            'exec {} -proc_id={} -syslog_level=7 -path={} -mode=PROD'.format(lom_plugin_mgr_path, proc_id, config_dir),
            'no shutdown',
            'exit',
        ]
        print ("start_command: ", start_command)
        start_response, error = switch_eapi.execute_command(start_command)
        if error:
            print("Error while starting lom-plmgr for proc_id {}: {}".format(proc_id, error))
            sys.exit(1)
        else:
            print("lom-plmgr started successfully for proc_id {}".format(proc_id) + "\n")
        
        # validate if lom-engine is running or not
        daemons_info, error = switch_eapi.extract_daemons_info()
        if error:
                print("Error while extracting daemon info:", error)
                sys.exit(1)

        print("Show daemon commmand output:")
        print_with_separator(json.dumps(daemons_info, indent=4))

        instance_name = "lom-plmgr-{}".format(proc_id)

        # Check if the instance is running
        print("Validating if {} is running or not".format(instance_name))
        running, error = switch_eapi.is_daemon_running(instance_name)
        if error:
            print("Error while checking {} status:".format(instance_name), error)
            sys.exit(1)
        else:
            if not running:
                print("{} is not running".format(instance_name))
                sys.exit(1)
            else:
                print("{} is running successfully".format(instance_name))
                
    # Final Validation if lom-engine and lom-plmgr are running or not
    daemons_info, error = switch_eapi.extract_daemons_info()
    if error:
        print("Error while extracting daemon info:", error)
        sys.exit(1)

    print("Show daemon commmand output:")
    print_with_separator(json.dumps(daemons_info, indent=4))

  # Check if lom-engine is running
    print("Validating if lom-engine is running or not")
    running, error = switch_eapi.is_daemon_running('lom-engine')
    if error:
        print("Error while checking lom-engine status:", error)
        sys.exit(1)
    else:
        if not running:
            print("lom-engine is not running")
            sys.exit(1)
        else:
            print("lom-engine is running successfully" + "\n")

    # Get information about all lom-plmgr instance
    print("Validating if lom-plmgr is running or not" + "\n")
    lom_plmgr_info, error = switch_eapi.get_daemon_lom_plmgr_info()
    if error:
        print("Error while getting lom-plmgr info:", error)
        sys.exit(1)
    else:
        # Chech the count of lom-plmgr instances
        if not lom_plmgr_info or len(lom_plmgr_info) != len(procs_keys):
            print("lom-plmgr is not running")
            sys.exit(1)
        else:
            print("lom-plmgr is running successfully")

    # agent uptimes validation 
    '''
    agent_uptimes_before, error = switch_eapi.get_agent_uptime_info()
    if error:
        print("Error while getting agent uptime info:", error)
        sys.exit(1)
    else:
        print("Agent uptime info before:")
        print_with_separator(agent_uptimes_before)

    time.sleep(30)

    agent_uptimes_after, error = switch_eapi.get_agent_uptime_info()
    if error:
        print("Error while getting agent uptime info:", error)
        sys.exit(1)
    else:
        print("Agent uptime info after wait:")
        print_with_separator(agent_uptimes_after)    

    # Validate if agent uptime is increased or not
    result, output = switch_eapi.compare_agent_uptimes(agent_uptimes_before, agent_uptimes_after)

    if result:
        print("Agent uptime is increased")
    else:
        print("Agent uptime is not increased")
        print(output)
        sys.exit(1)
    '''
   # core dump validation
    '''
    core_dump_info_before, error = switch_eapi.get_system_coredump()
    if error:
        print("Error while getting core dump info:", error)
        sys.exit(1)
    else:
        print("Core dump info before:")
        print_with_separator(json.dumps(core_dump_info_before, indent=4))

    
    # Wait for 30 seconds
    #time.sleep(30)

    core_dump_info_after, error = switch_eapi.get_system_coredump()
    if error:
        print("Error while getting core dump info:", error)
        sys.exit(1)
    else:
        print("Core dump info after wait:")
        print_with_separator(json.dumps(core_dump_info_after, indent=4))

    # Validate if core dump is generated or not
    result, output = switch_eapi.compare_coredump(core_dump_info_before, core_dump_info_after)

    if result:
        print("Core dump matched")
        print(output)
    else:
        print("Core dump is not matched")
        print(output)
        sys.exit(1)
    '''
    '''
    # hard capacity utilization validation
    capacity_before, error = switch_eapi.get_hardware_capacity_utilization()
    if error:
        print("Error while getting capacity info:", error)
        sys.exit(1)
    else:
        print("Capacity info before:")
        print_with_separator(json.dumps(capacity_before, indent=4))

    # Wait for 30 seconds
    #time.sleep(30)

    capacity_after, error = switch_eapi.get_hardware_capacity_utilization()
    if error:
        print("Error while getting capacity info:", error)
        sys.exit(1)
    else:
        print("Capacity info after wait:")
        print_with_separator(json.dumps(capacity_after, indent=4))

    # Validate if capacity is increased or not
    result, output = switch_eapi.compare_capacity_utilization(capacity_before, capacity_after, 5)
    if result:
        print("Capacity is stable")
        print(output)
    else:
        print("Capacity is increased")
        print(output)
        sys.exit(1)
    '''

except Exception as e:
    print("An unexpected error occurred: {0}".format(e))
    sys.exit(1)



