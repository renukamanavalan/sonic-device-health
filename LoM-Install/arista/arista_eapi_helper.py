from __future__ import print_function  # Compatibility for Python 2 and 3
import os
import json
import sys
import re
import argparse
import jsonrpclib

'''
Uses the jsonrpclib library to connect to the switch via EAPI.
'''

class AristaSwitchEAPIHelper(object):
    """
    This class provides a helper for interacting with an Arista switch via EAPI.
    """

    def __init__(self):
        """
        Initialize the AristaSwitchEAPIHelper instance.
        """
        self.server = None

    def connect(self, socket_path='/var/run/command-api.sock'):
        """
        Connect to the Arista switch via EAPI.

        Args:
            socket_path (str): The path to the Unix socket for EAPI connection. Defaults to '/var/run/command-api.sock'.
        """
        try:
            # Format the URL for Unix socket connection
            url = 'unix://./{}'.format(socket_path)
            self.server = jsonrpclib.Server(url)
        except Exception as e:
            raise Exception("Error connecting to the switch: {0}".format(str(e)))    

    def execute_command(self, command):
        """
        Execute a command on the Arista switch via EAPI.

        Args:
            command (str): The command to be executed. Its array of words must be separated by spaces.

        Returns:
            tuple: A tuple containing the response from the switch and any error message.
        """
        if self.server is None:
            return None, "Error: Not connected to the switch."
        try:
            response = self.server.runCmds(1, command)
            return response, None
        except jsonrpclib.ProtocolError as e:
            return None, "Protocol error: {}".format(e)
        except Exception as e:
            return None, "An error occurred while executing the command: {}".format(e)

    # Usage
    #switch_eapi = AristaSwitchEAPIHelper()
    #switch_eapi.connect(socket_path='/var/run/command-api.sock')  # Use your specific socket path
    #switch_eapi.execute_command("show daemon")

    '''
    show daemon output returned by the switch via jsonrpclib:
       [
                {
                    "daemons": {
                        "lom-engine": {
                            "pid": 1234,
                            "uptime": 123456,
                            "starttime": "2019-01-01T00:00:00",
                            "running": true
                        },
                        "lom-plmgr-proc_0": {
                            "pid": 1234,
                            "uptime": 123456,
                            "starttime": "2019-01-01T00:00:00",
                            "running": true
                        },
                        "lom-plmgr-proc_1": {
                            "pid": 1234,
                            "uptime": 123456,
                            "starttime": "2019-01-01T00:00:00",
                            "running": true
                        }
                    }
                }
            ]
        or 

       [
                {
                    "daemons": {}
                }
        ]
    '''
    def extract_daemons_info(self):
        """
        Execute 'show daemon' command and extract process information from the JSON output.
        """
        processes = {}
        error = None  # Initialize the error flag to None

        try:
            # Execute 'show daemon' command
            daemon_command = ['show daemon']
            show_daemon_output, error = self.execute_command(daemon_command)

            if error:
                return None, "Error while executing show daemon command: {}".format(error)

            if show_daemon_output:
                daemons = show_daemon_output[0].get("daemons", {})
                
                if not daemons:  # Check if daemons dictionary is empty
                    return processes, None  # Return an empty processes dictionary and no error
                
                for process_name, process_info in daemons.items():
                    processes[process_name] = {
                        "PID": process_info.get("pid", None),
                        "Uptime": process_info.get("uptime", None),
                        "StartTime": process_info.get("starttime", None),
                        "Running": process_info.get("running", False),
                    }
        except Exception as e:
            error = str(e)  # Store the exception message in the error flag

        return processes, error  # Return the processes and the error flag

    def is_daemon_running(self, daemon_name):
        """
        Check if a specific daemon is running.

        Parameters:
        - daemon_name: The name of the daemon to check.

        Returns:
        - running: Boolean indicating whether the daemon is running.
        - error: Error message if any.
        """
        processes, error = self.extract_daemons_info()
        if error:
            return None, error  # Return None and the error message

        daemon_info = processes.get(daemon_name, {})
        running = daemon_info.get("Running", False)

        return running, None  # Return the running status and no error

    def is_daemon_config_exists(self, daemon_name):
        """
        Check if a specific daemon's configuration exists.

        Parameters:
        - daemon_name: The name of the daemon to check.

        Returns:
        - exists: Boolean indicating whether the daemon's configuration exists.
        - error: Error message if any.
        """
        processes, error = self.extract_daemons_info()
        if error:
            return None, error  # Return None and the error message

        daemon_info = processes.get(daemon_name, {})
        exists = len(daemon_info) > 0  # If daemon_info is not empty, the configuration exists

        return exists, None  # Return the existence status and no error
    
    def disable_daemon(self, daemon_name):
        """
        This function disables a specified daemon.

        Parameters:
        - instance_name: The name of the daemon to disable.

        Returns:
        - result: The result of the command execution.
        - error: Error message if any.
        """

        # Define the command sequence to disable the daemon
        command = [
            'configure',
            'no daemon {}'.format(daemon_name),
            'exit',
        ]

        try:
            result, error = self.execute_command(command)

            if error:
                return None, "Error: Failed to execute command. {}".format(error)
        except Exception as e:
            return None, "Error: Failed to disable daemon. {}".format(e)

        return result, None
    
    def get_daemon_lom_engine_info(self):
        """
        Execute 'show daemon' command and check if the 'lom-engine' process is running.
        """
        processes, error = self.extract_daemons_info()
        if error:
            return None, error  # Return None and the error message
        lom_engine_info = processes.get("lom-engine", {})

        return lom_engine_info, None  # Return the lom_engine_info and no error

    def get_daemon_lom_plmgr_info(self):
        """
        Execute 'show daemon' command and check if the 'lom-plmgr' process is running.
        """
        processes, error = self.extract_daemons_info()
        if error:
            return None, error  # Return None and the error message

        lom_plmgr_info = {k: v for k, v in processes.items() if k.startswith("lom-plmgr")}

        return lom_plmgr_info, None  # Return the lom_plmgr_info dictionary and no error       

    def get_agent_uptime_info(self):
        """
        Run 'show agent uptime' command and parse the output to return a dictionary of agent uptimes.
        Sample  'show agent uptime' output format :
        [
                {
                    "agents": {
                        "lldp": {
                            "agentStartTime": 123456,
                            "restartCount": 0
                        },
                        "stp": {
                            "agentStartTime": 123456,
                            "restartCount": 0
                        },
                        "dhcp_relay": {
                            "agentStartTime": 123456,
                            "restartCount": 0
                        },
                        "dhcp_relaySyncd
        
        Parameters:
        - None

        Returns:
        - agent_uptimes: Dictionary of agent uptimes.
        - error: Error message if any.

        Output Format :
        {
            "lldp": {
                "AgentStartTime": 123456,
                "RestartCount": 0
            },
            "stp": {
                "AgentStartTime": 123456,
                "RestartCount": 0
            }
        }
        """
        show_agent_uptime_command = ['show agent uptime']
        show_agent_uptime_output, error = self.execute_command(show_agent_uptime_command)

        if error:
            return None, error  # Return None and the error message

        agent_uptimes, error = self.parse_agent_uptime_output(show_agent_uptime_output)
        
        if error:
            return None, error  # Return None and the error message

        return agent_uptimes, None  # Return the agent uptimes and no error
         
 
    def parse_agent_uptime_output(self, show_agent_uptime_output):
        """
        Parse the JSON 'show agent uptime' command output and return a dictionary of agent uptimes.

        Parameters:
        - show_agent_uptime_output: JSON output of 'show agent uptime' command.

        Returns:
        - agent_uptimes: Dictionary of agent uptimes.
        - error: Error message if any.

        Output Format :
        {
            "lldp": {
                "AgentStartTime": 123456,
                "RestartCount": 0
            },
            "stp": {
                "AgentStartTime": 123456,
                "RestartCount": 0
            }
        }
        """
        agent_uptimes = {}
        error = None  # Initialize the error flag to None

        if not show_agent_uptime_output:
            return agent_uptimes, None

        try:
            agent_info = show_agent_uptime_output[0].get("agents", {})
            for agent_name, agent_data in agent_info.items():
                agent_uptimes[agent_name] = {
                    'AgentStartTime': agent_data.get("agentStartTime", None),
                    'RestartCount': agent_data.get("restartCount", None)
                }
        except Exception as e:
            error = str(e)  # Store the exception message in the error flag

        return agent_uptimes, error  # Return the agent uptimes and the error flag
    
    def compare_agent_uptimes(self, agent_uptimes_first, agent_uptimes_second):
        """
        Compare two sets of agent uptimes based on the specified conditions.

        Parameters:
        - agent_uptimes_first: First set of agent uptimes. Format is the same as the output of parse_agent_uptime_output().
        - agent_uptimes_second: Second set of agent uptimes. Format is the same as the output of parse_agent_uptime_output().

        Returns:
        - comparison_result: Boolean indicating whether all conditions are met.
        - error_output: Dictionary with error messages for each agent.

        Output Format :
        {
            true, {}
        }

        or 

        {
            false, {
                "lldp": [
                    "AgentStartTime is less than the first set of uptimes"
                ],
                "stp": [
                    "AgentStartTime is less than the first set of uptimes",
                    "RestartCount does not match"
                ]
            }
        }
        """
        comparison_result = True
        error_output = {}

        for agent_name, uptime1 in agent_uptimes_first.items():
            uptime2 = agent_uptimes_second.get(agent_name, None)

            if uptime2 is None:
                comparison_result = False
                error_output[agent_name] = ["Agent not found in the second set of uptimes"]
            else:
                agent_errors = []

                if uptime2['AgentStartTime'] != uptime1['AgentStartTime']:
                    agent_errors.append("AgentStartTime is less than the first set of uptimes")
                    comparison_result = False

                if uptime1['RestartCount'] != uptime2['RestartCount']:
                    agent_errors.append("RestartCount does not match")
                    comparison_result = False

                if agent_errors:
                    error_output[agent_name] = agent_errors

        return comparison_result, error_output

    # This returns error in EOS 4.21 
    def get_system_coredump(self) :
        """
        Run the 'show system coredump | json' command and return the core dump information.

        Parameters:
        - None

        Returns:
        - core_dump_info: Core dump information in JSON format.

        Output Format :
        {
            "mode": "compressed",
            "coreFiles": []
        }
        """
        command = ['show system coredump']
        core_dump_info, error = self.execute_command(command)
        if error:
            return None, error        
        core_dump_info = core_dump_info[0]
        return core_dump_info, None  # Return the core dump information and no error
    
    '''

        Sample coredump outputs:

        [
            {
                "mode": "compressed", 
                "coreFiles": [
                    "core.11535.1699053220.vim.gz", 
                    "core.11488.1699053207.vim.gz", 
                    "core.11184.1699053078.vim_n.gz", 
                    "core.8716.1699051902.vim.gz", 
                    "core.8050.1699051619.vim.gz", 
                    "core.7978.1699051603.vim.gz"
                ]
            }
        ]

        or 
        [
            {
                "mode": "compressed",
                "coreFiles": []
            }
        ]

        or 

        [
            {
                "errors": [
                    "This is an unconverted command"
                ]
            }
        ]

    '''
    def compare_coredump(self, core_dump_info1, core_dump_info2):
        """
        Compare two sets of core dump information and return a boolean indicating if they match.

        Parameters:
        - core_dump_info1: First set of core dump information.
        - core_dump_info2: Second set of core dump information.

        Returns:
        - match: True if the coreFiles match, False otherwise.
        - unmatched_corefiles: I false, List of coreFiles that do not match (if there are differences).
        -          If true, List of coreFiles in the first set.
        """
        core_files1 = set(core_dump_info1.get('coreFiles', []))
        core_files2 = set(core_dump_info2.get('coreFiles', []))
        #print_with_separator("Core files in the first set: {0}".format(core_files1))
        #print_with_separator("Core files in the second set: {0}".format(core_files2))
        # Check if the coreFiles match
        match = core_files1 == core_files2

        if not match:
            # Find the unmatched coreFiles
            unmatched_corefiles = list(core_files1.symmetric_difference(core_files2))
            return False, unmatched_corefiles

        return True, list(core_files1)

    def get_hardware_capacity_utilization(self, peercentage_threshold=0):
        """
        Run the 'show hardware capacity utilization percent exceed <peercentage_threshold> | json' command and return the 'tables' output.

        command output:
        STR-ODL-7060CX-01(config)#show hardware capacity utilization percent exceed 0 | json 
        {
            "tables": [
                {
                    "highWatermark": 0,
                    "used": 0,
                    "usedPercent": 0,
                    "committed": 0,
                    "table": "VFP",
                    "chip": "Linecard0/0",
                    "maxLimit": 256,
                    "feature": "Slice-3",
                    "free": 256
                },
                {
                    "highWatermark": 55,
                    "used": 55,
                    "usedPercent": 21,
                    "committed": 0,
                    "table": "IFP",
                    "chip": "Linecard0/0",
                    "maxLimit": 256,
                    "feature": "Slice-0",
                    "free": 201
                }
            ]
        }

        Parameters:
        - peercentage_threshold: Percentage threshold for capacity utilization.

        Returns:
        - output: 'tables' output in JSON format.
        - error: Error message if any.

        FUnction Output Format :
        {
            "VFP$Slice-3$Linecard0/0": 56,
            "IFP$Slice-0$Linecard0/0": 20
        }

        """

        command = ['show hardware capacity utilization percent exceed {0} | json'.format(peercentage_threshold)]
        tables_output, error = self.execute_command(command)
        if error:
            return None, error # Return None and the error message
        
        tables_output = tables_output[0].get('tables', [])
        output, error = self.parse_capacity_utilization(tables_output)

        if error:
            return None, error
        
        return output, None  # Return the tables output and no error
    
    def parse_capacity_utilization(self, tables_json):
        """
        Parse capacity utilization information and generate a dictionary with unique keys.

        Parameters:
        - tables_json: List of tables with capacity utilization information.

        Returns:
        - utilization_dict: Dictionary with keys formed from 'Table, Feature, and Chip' and values as 'usedPercent'.
        - error: Error message if an exception occurs.

        Output Format :
        {
            "VFP$Slice-3$Linecard0/0": 56,
            "IFP$Slice-0$Linecard0/0": 20
        }

        """
        utilization_dict = {}
        error = None

        try:
            for table_info in tables_json:
                table = table_info['table']
                feature = table_info['feature']
                chip = table_info['chip']
                usedPercent = table_info['usedPercent']

                key = "{}${}${}".format(table, feature, chip)
                utilization_dict[key] = usedPercent

        except Exception as e:
            error = "An error occurred while parsing capacity utilization: {}".format(e)

        return utilization_dict, error
    
    def compare_capacity_utilization(self, tables_json_before, tables_json_after, threshold=1):
        """
        Compare two sets of capacity utilization information based on the percentage threshold.

        Parameters:
        - tables_json_before: Dictionary with keys representing Table, Feature, and Chip, and values as usedPercent before.
        - tables_json_after: Dictionary with keys representing Table, Feature, and Chip, and values as usedPercent after.
        - threshold: Percentage threshold for the difference.

        Returns:
        - comparison_result: Boolean indicating whether all comparisons meet the threshold.
        - error_output: Dictionary with error messages for each key.

        Output Format :
        {
            true, {}
        }

        or 

        {
            false, {
                "VFP$Slice-3$Linecard0/0": [
                    "usedPercent is empty post check"
                ],
                "IFP$Slice-0$Linecard0/0": [
                    "Percentage difference 4 is greater than threshold 1."
                ]
            }
        }
        """
        comparison_result = True
        error_output = {}

        for key, used_percent_before in tables_json_before.items():
            used_percent_after = tables_json_after.get(key, None)

            if used_percent_after is None:
                comparison_result = False
                error_output[key] = ["usedPercent is empty post check"]
                continue

            percentage_diff = abs(used_percent_before - used_percent_after)
            if percentage_diff > threshold:
                comparison_result = False
                if key not in error_output:
                    error_output[key] = []
                error_output[key].append("Percentage difference {} is greater than threshold {}.".format(percentage_diff, threshold))

        return comparison_result, error_output
    

    def set_hardware_drop_counter_iptcrc(self, chipname, counter_value):
        """
        This function sets the hardware drop counter 'Ipt0CrcErrCnt' for a specific chip.

        Parameters:
        - chipname: The name of the chip for which the counter is to be set.
        - counter_value: The value to which the counter is to be set.

        Returns:
        - result: The result of the command execution.
        - error: Error message if any.
        """

        # Define the command sequence to set the counter
        command = [
            'configure',  # Enter configuration mode
            'platform fap {} counters set Ipt0CrcErrCnt {}'.format(chipname, counter_value),  # Set the counter for the specified chip
            'exit',  # Exit configuration mode
        ]

        try:
            result, error = self.execute_command(command)

            if error:
                return None, "Error: Failed to execute command. {}".format(error)
        except Exception as e:
            return None, "Error: Failed to set counter. {}".format(e)

        # If the command sequence executes successfully, return the result and None for the error
        return result, None

    def show_hardware_counter_drop_count(self, chipname, counter):
        """
        This function retrieves the drop count for a specific hardware counter on a specific chip.

        Parameters:
        - chipname: The name of the chip for which the counter drop count is to be retrieved.
        - counter: The name of the counter for which the drop count is to be retrieved.

        Returns:
        - dropCount: The drop count for the specified counter on the specified chip.
        - error: Error message if any.

        The function executes the 'show hardware counter drop' command and parses the output to find the drop count for the specified counter on the specified chip. If the counter is not found on the chip, the function returns 0 for the drop count. If an error occurs during command execution or an exception is raised, the function returns None for the drop count and an error message.

        Output sample:
        {
            "totalPacketProcessorDrops": 64,
            "totalCongestionDrops": 0,
            "totalAdverseDrops": 1,
            "dropEvents": {
                "Jericho6/0": {
                    "dropEvent": [
                        {
                            "lastEventTime": "2024-01-09 16:49:12",
                            "eventCount": 1,
                            "dropInLastMinute": 0,
                            "initialEventTime": "2024-01-09 16:49:12",
                            "dropInLastOneDay": 0,
                            "dropInLastOneHour": 0,
                            "dropInLastTenMinute": 0,
                            "dropCount": 1,
                            "counterType": "PacketProcessor",
                            "counterId": 48,
                            "counterName": "dropVoqInMcastEmptyMcid"
                        }
                    ]
                },
                ...
            }
        }
        """

        # Define the command to show the hardware counter drop
        command = ['show hardware counter drop']

        try:
            output, error = self.execute_command(command)
            if error:
                return None, "Error: Failed to execute command. {}".format(error)

            # Parse the command output
            for chip, data in output[0]['dropEvents'].items():
                if chip == chipname:
                    for event in data['dropEvent']:
                        if event['counterName'] == counter:
                            return event['dropCount'], None

            # If the counter is not found on the chip, return 0 for the drop count and None for the error
            return 0, None
        except Exception as e:
            return None, "Error: Failed to show counter drop count. {}".format(e)

    def enable_terminattr_daemon(self, restaddr='127.0.0.1', restport='6040', grpcaddr='0.0.0.0', grpcport='5910', namespace=None):
        """
        This function enables the TerminAttr daemon.

        Parameters:
        - restaddr: The IP address for the REST server. Default is '127.0.0.1'.
        - restport: The port for the REST server. Default is '6040'.
        - grpcaddr: The IP address for the gRPC server. Default is '0.0.0.0'.
        - grpcport: The port for the gRPC server. Default is '5910'.
        - namespace: The namespace in which the gRPC server should run. Optional.

        Returns:
        - result: The result of the command execution.
        - error: Error message if any.
        """

        # Define the gRPC address
        grpc_addr = '{}:{}'.format(grpcaddr, grpcport) if namespace is None else '{}/{}:{}'.format(namespace, grpcaddr, grpcport)

        # Define the command sequence to enable TerminAttr
        command = [
            'configure', 
            'daemon TerminAttr', 
            'exec /usr/bin/TerminAttr -restaddr={}:{} -disableaaa -grpcaddr={}'.format(restaddr, restport, grpc_addr), 
            'shutdown',  
            'no shutdown', 
            'exit',
        ]

        try:
            result, error = self.execute_command(command)

            if error:
                return None, "Error: Failed to execute command. {}".format(error)
        except Exception as e:
            return None, "Error: Failed to enable TerminAttr. {}".format(e)

        return result, None

    def get_daemon_terminattr_info(self):
        """
        Execute 'show daemon' command and check if the 'TerminAttr' process is running.
        """
        processes, error = self.extract_daemons_info()
        if error:
            return None, error  # Return None and the error message

        terminattr_info = {k: v for k, v in processes.items() if k.startswith("TerminAttr")}

        return terminattr_info, None 
    
    def check_and_enable_terminattr_daemon(self, restaddr='127.0.0.1', restport='6040', grpcaddr='0.0.0.0', grpcport='5910', namespace=None):
        """
        Check if the TerminAttr daemon is running, and if not, try to enable it.

        Parameters:
        - restaddr: The IP address for the REST server. Default is '127.0.0.1'.
        - restport: The port for the REST server. Default is '6040'.
        - grpcaddr: The IP address for the gRPC server. Default is '0.0.0.0'.
        - grpcport: The port for the gRPC server. Default is '5910'.
        - namespace: The namespace in which the gRPC server should run. Optional.

        Returns:
        - running: Boolean indicating whether the TerminAttr daemon is running.
        - already_enabled: Boolean indicating whether the TerminAttr daemon was already enabled.
        - error: Error message if any.
        """
        running, error = self.is_daemon_running("TerminAttr")
        already_enabled = running
        if not running and not error:
            # TerminAttr daemon is not running, try to enable it
            running, error = self.enable_terminattr_daemon(restaddr, restport, grpcaddr, grpcport, namespace)
        return running, already_enabled, error

def main():
    parser = argparse.ArgumentParser(description='Arista Switch EAPI Helper')
    parser.add_argument(
        '--api', 
        choices=[
            'execute_command', 
            'extract_daemons_info',
            'is_daemon_running',
            'is_daemon_config_exists',
            'disable_daemon',
            'get_daemon_lom_engine_info',
            'get_daemon_lom_plmgr_info',
            'get_agent_uptime_info',
            'get_system_coredump', 
            'get_hardware_capacity_utilization',    
            'set_hardware_drop_counter_iptcrc', 
            'show_hardware_counter_drop_count',    
            'enable_terminattr_daemon',
            'get_daemon_terminattr_info',
            'check_and_enable_terminattr_daemon'                         
        ], 
        help='Select API to run with proper arguments. Comma seperated for config commands'
    )
    parser.add_argument('--command', help='CLI Command to execute')
    parser.add_argument('--chipname', help='Chip name for set_hardware_drop_counter_iptcrc and show_hardware_counter_drop_count command')
    parser.add_argument('--counter', help='Counter value for set_hardware_drop_counter_iptcrc and counter name show_hardware_counter_drop_count command')
    parser.add_argument('--daemon_name', help='Daemon name for is_daemon_running command, disable_daemon command and is_daemon_config_exists command')

    args = parser.parse_args()

    # Check if any arguments were provided
    if not any(vars(args).values()):
        parser.print_help()
        return

    if args.api == 'execute_command' and not args.command:
        parser.error("--command is required with 'execute_command' API")

    if args.api in ['set_hardware_drop_counter_iptcrc', 'show_hardware_counter_drop_count'] and (not args.chipname or not args.counter):
        parser.error("--chipname and --counter are required with 'set_hardware_drop_counter_iptcrc' and 'show_hardware_counter_drop_count' APIs")

    arista_manager = AristaSwitchEAPIHelper()

    try:
        arista_manager.connect()
    except Exception as e:
        print("Error: Failed to connect. {}".format(e))
        return

    if args.api == 'execute_command':
        try:
            command = re.sub(' +', ' ', args.command).strip()
            command_list = command.split(',')
            command_list = [cmd.strip() for cmd in command_list]
            result, error = arista_manager.execute_command(command_list)
            if error:
                print("Error: Failed to execute command. {}".format(error))
                return
            print("Result: {}".format(json.dumps(result, indent=4)))
        except Exception as e:
            print("Error: Failed to execute command. {}".format(e))
    elif args.api == 'extract_daemons_info':
        try:
            result, error = arista_manager.extract_daemons_info()
            if error:
                print("Error: Failed to extract daemons info. {}".format(error))
                return
            print("Result: {}".format(result))
        except Exception as e:
            print("Error: Failed to extract daemons info. {}".format(e))
    elif args.api == 'is_daemon_running':
        try:
            result, error = arista_manager.is_daemon_running(args.daemon_name)
            if error:
                print("Error: Failed to check if daemon is running. {}".format(error))
                return
            print("Result: {}".format(result))
        except Exception as e:
            print("Error: Failed to check if daemon is running. {}".format(e))
    elif args.api == 'is_daemon_config_exists':
        try:
            result, error = arista_manager.is_daemon_config_exists(args.daemon_name)
            if error:
                print("Error: Failed to check if daemon config exists. {}".format(error))
                return
            print("Result: {}".format(result))
        except Exception as e:
            print("Error: Failed to check if daemon config exists. {}".format(e))
    elif args.api == 'disable_daemon':
        try:
            result, error = arista_manager.disable_daemon(args.daemon_name)
            if error:
                print("Error: Failed to disable daemon. {}".format(error))
                return
            print("Successfully disabled daemon.")
        except Exception as e:
            print("Error: Failed to disable daemon. {}".format(e))
    elif args.api == 'get_daemon_lom_engine_info':
        try:
            result, error = arista_manager.get_daemon_lom_engine_info()
            if error:
                print("Error: Failed to get lom engine info. {}".format(error))
                return
            print("Result: {}".format(result))
        except Exception as e:
            print("Error: Failed to get lom engine info. {}".format(e))
    elif args.api == 'get_daemon_lom_plmgr_info':
        try:
            result, error = arista_manager.get_daemon_lom_plmgr_info()
            if error:
                print("Error: Failed to get lom plmgr info. {}".format(error))
                return
            print("Result: {}".format(result))
        except Exception as e:
            print("Error: Failed to get lom plmgr info. {}".format(e))
    elif args.api == 'set_hardware_drop_counter_iptcrc':
        try:
            result, error = arista_manager.set_hardware_drop_counter_iptcrc(args.chipname, args.counter)
            if error:
                print("Error: Failed to set hardware drop counter. {}".format(error))
                return
            print("Successfully set hardware drop counter.")
        except Exception as e:
            print("Error: Failed to set hardware drop counter. {}".format(e))
    elif args.api == 'show_hardware_counter_drop_count':
        try:
            result, error = arista_manager.show_hardware_counter_drop_count(args.chipname, args.counter)
            if error:
                print("Error: Failed to show counter drop count. {}".format(error))
                return
            print("Drop count for counter '{}' on chip '{}' is {}.".format(args.counter, args.chipname, result))
        except Exception as e:
            print("Error: Failed to show counter drop count. {}".format(e))
    elif args.api == 'get_system_coredump':
        try:
            result, error = arista_manager.get_system_coredump()
            if error:
                print("Error: Failed to get system coredump. {}".format(error))
                return
            print("Result: {}".format(result))
        except Exception as e:
            print("Error: Failed to get system coredump. {}".format(e))
    elif args.api == 'get_hardware_capacity_utilization':
        try:
            result, error = arista_manager.get_hardware_capacity_utilization()
            if error:
                print("Error: Failed to get hardware capacity utilization. {}".format(error))
                return
            print("Result: {}".format(result))
        except Exception as e:
            print("Error: Failed to get hardware capacity utilization. {}".format(e))
    elif args.api == 'get_agent_uptime_info':
        try:
            result, error = arista_manager.get_agent_uptime_info()
            if error:
                print("Error: Failed to get agent uptime info. {}".format(error))
                return
            print("Result: {}".format(result))
        except Exception as e:
            print("Error: Failed to get agent uptime info. {}".format(e))
    elif args.api == 'enable_terminattr_daemon':
        try:
            result, error = arista_manager.enable_terminattr_daemon()
            if error:
                print("Error: Failed to enable TerminAttr. {}".format(error))
                return
            print("Successfully enabled TerminAttr.")
        except Exception as e:
            print("Error: Failed to enable TerminAttr. {}".format(e))
    elif args.api == 'get_daemon_terminattr_info':
        try:
            result, error = arista_manager.get_daemon_terminattr_info()
            if error:
                print("Error: Failed to get TerminAttr info. {}".format(error))
                return
            print("Result: {}".format(result))
        except Exception as e:
            print("Error: Failed to get TerminAttr info. {}".format(e))
    elif args.api == 'check_and_enable_terminattr_daemon':
        try:
            running, already_enabled, error = arista_manager.check_and_enable_terminattr_daemon()
            if error:
                print("Error: Failed to check and enable TerminAttr. {}".format(error))
                return
            print("TerminAttr daemon is currently: {}".format("Running" if running else "Not running"))
            print("TerminAttr daemon was already enabled: {}".format(already_enabled))
        except Exception as e:
            print("Error: Failed to check and enable TerminAttr. {}".format(e))

if __name__ == "__main__":
    main()