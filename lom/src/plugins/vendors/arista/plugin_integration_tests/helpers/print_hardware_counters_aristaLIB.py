"""
This tool is used to print the hardware counters for Arista switches in a readable format.
It can be used to print all counters or a specific counter.
The tool can be used in two ways:
1. Get the counters once and print them. This is the default method. Counter details are printed in a readable format
2. Subscribe to the counters and print them as they are updated. Counter details are printed in a raw format

The tool requires the gnmi_arista executable to be in the same directory.
The tool also requires the gnmi_arista executable to be executable.
The tool can be run from any directory.

Usage:
    python print_hardware_counters_aristaLIB.py --gnmi_arista <path_to_gnmi_arista> --addr <address> --username <username> --password <password> --counterName <counter_name> --method <method>

    --gnmi_arista: Full path to the gnmi_arista executable.
    --addr: Address of the switch in the format <ip>:<port>.
    --username: Username to use to connect to the switch.
    --password: Password to use to connect to the switch.
    --counterName: Name of the counter to print. If not specified, all counters will be printed.
    --method: Method to use to get the counters. If not specified, the default method is get.
        get: Get the counters once and print them.
        subscribe: Subscribe to the counters and print them as they are updated.

Examples:
    python print_hardware_counters_aristaLIB.py --gnmi_arista ./gnmi_arista --addr localhost:5910 --username admin --password password --counterName all --method get
    
"""

import subprocess
import json
import re
from collections import defaultdict
from collections import deque
import sys
import argparse
import os

# Define the mapping for counter IDs
counter_id_mapping = {
    1: "IptCrcErrCnt",
    2: "UcFifoFullDrop",
    3: "UcFifoSnoopDrop",
    4: "UcFifoMirrorDrop",
    5: "IngrReplFifoDrop",
    6: "IngrReplFifoSnoopDrop",
    7: "IngrReplFifoMirrorDrop",
    8: "DeqDeletePktCnt",
    9: "RqpDiscardPacketCtr",
    10: "RqpPC",
    11: "McastDataBuffersDropCtr",
    12: "PqpMcastPC",
    13: "EnqPktCnt",
    14: "IngressDiscardCounter",
    15: "EgressDiscardCounter",
    16: "FabricReceiveCellDiscard",
    17: "SerdesCrcErrors",
    18: "DchFifosDiscards",
    19: "DchReordDiscards",
    20: "DchUnreachables",
    21: "DchFifosMulticastLowDrops",
    22: "RtpDrhMulticastLowDrops",
    23: "DcmFifosIpDrops",
    24: "DcmFifosTotalDrops",
    25: "DclFifosTotalDrops",
    26: "dropVoqInPortNotVlanMember",
    27: "dropVoqInSaEqualsDa",
    28: "dropVoqInSaMulticast",
    29: "dropVoqInRpf",
    30: "dropVoqInIpv4VersionError",
    31: "dropVoqInIpv4ChecksumError",
    32: "dropVoqInIpv6VersionError",
    33: "dropVoqInIpv6UnspecifiedDestination",
    34: "dropVoqInIpv6MulticastSource",
    35: "dropVoqInAcl",
    36: "dropVoqInNoArp",
    37: "dropVoqInNullRoute",
    38: "dropVoqInRouteOverflow",
    39: "dropVoqInPbr",
    40: "dropVoqInTunRoute",
    41: "dropVoqInTunGre",
    42: "dropVoqInTunMpls",
    43: "dropVoqInTunVxlan",
    44: "dropVoqInMcastNoCpu",
    45: "dropVoqInElkError",
    46: "EgressAclDropCounter",
    47: "dropVoqInMcastEmptyMcid",
    48: "EgrMroutedTtlDrop",
    49: "dropVoqInForwardingLookupMiss",
    50: "dropVoqInNoForwardingAction"
}

# Add these lines to define current_chip_id and current_counter_id in the global scope
current_chip_id = None
current_counter_id = None

def bytes_to_string(bytes_list):
    return ''.join(chr(b) for b in bytes_list if b != 0)

def is_valid_json(json_string):
    try:
        json.loads(json_string)
        return True
    except ValueError:
        return False

def parse_output(output):
    if not output:
        print("Error: No output to parse.")
        sys.exit(1)

    parsed_data_dict = defaultdict(lambda: defaultdict(dict))
    queue = deque()
    lines = output.split('\n')
    skip_next_value = False

    for line in lines:
        if line.startswith('/Smash/hardware/counter/internalDrop/SandCounters/_counts/internalDrop:'):
            skip_next_value = True
            continue
        elif line.startswith('/Smash/hardware/counter/internalDrop/SandCounters/internalDrop/'):
            skip_next_value = False
            # If there are items in the queue, assign them to the corresponding parameter
            while queue:
                param, value = queue.popleft()
                chip_id, chip_type, counter_id = extract_ids(param)
                if not all([chip_id, chip_type, counter_id]):
                    print("Error: Failed to extract ids from path.")
                    sys.exit(1)
                parsed_value = parse_value(value)
                param = param.split('/')[-1].rstrip(':')
                parsed_data_dict[chip_id][counter_id][param] = parsed_value

            # Extract the parameter name and store it in the queue
            queue.append((line, []))
        elif not skip_next_value:
            # The line is a value, store it in the queue
            param, value = queue.pop()
            value.append(line.strip())
            queue.append((param, value))

    # If there are items left in the queue, assign them to the corresponding parameter
    while queue:
        param, value = queue.popleft()
        chip_id, chip_type, counter_id = extract_ids(param)
        if not all([chip_id, chip_type, counter_id]):
            print("Error: Failed to extract ids from path.")
            sys.exit(1)
        parsed_value = parse_value(value)
        param = param.split('/')[-1].rstrip(':')
        parsed_data_dict[chip_id][counter_id][param] = parsed_value

    # Validate the extracted ids and chip_type
    for chip_id, counters in parsed_data_dict.items():
        for counter_id, params in counters.items():
            validate_ids(chip_id, chip_type, counter_id, params)

    return parsed_data_dict

def parse_value(value_lines):
    value_str = ''.join(value_lines)
    if value_str.startswith('[') and value_str.endswith(']'):
        ascii_values = json.loads(value_str)
        return ''.join(chr(val) for val in ascii_values)
    elif is_valid_json(value_str):
        json_value = json.loads(value_str)
        if isinstance(json_value, dict) and 'value' in json_value:
            return json_value['value']
        else:
            return json_value
    else:
        return value_str

def extract_ids(path):
    match = re.search(r'/(\d+)_([a-zA-Z]+)_(\d+)_', path)
    if match:
        chip_id = match.group(1)
        chip_type = match.group(2)
        counter_id = match.group(3)
        return chip_id, chip_type, counter_id
    return None, None, None

def validate_ids(chip_id, chip_type, counter_id, params):
    if 'chipId' in params and int(params['chipId']) != int(chip_id):
        print("Error: chipId from gNMI path ({}) does not match chipId from parameters ({}).".format(chip_id, params['chipId']))
        sys.exit(1)
    if 'counterId' in params and int(params['counterId']) != int(counter_id):
        print("Error: counterId from gNMI path ({}) does not match counterId from parameters ({}).".format(counter_id, params['counterId']))
        sys.exit(1)
    if 'chipType' in params and params['chipType'] != chip_type:
        print("Error: chipType from parameters ({}) does not match chipType from gNMI path ({}).".format(params['chipType'], chip_type))
        sys.exit(1)

def run_command_get(command):
    try:
        output = subprocess.check_output(command, shell=True)
        output = output.decode('utf-8')
        return output
    except subprocess.CalledProcessError as e:
        print("An error occurred while running the command.")
        print(e.output)
        sys.exit(1)

def run_command_subscribe(command):
    try:
        process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        while True:
            output = process.stdout.readline()
            if output == '' and process.poll() is not None:
                break
            if output:
                print(output.strip())
        rc = process.poll()
        return rc
    except subprocess.CalledProcessError as e:
        print("An error occurred while running the command.")
        print(e.output)
        sys.exit(1)

def print_table(data, counter_name):
    print("Total number of chips: {}, countername : {}\n".format(len(data), counter_name))
    for chip_id, counters in data.items():
        for counter_id, params in counters.items():            
            if counter_name == 'all' or params.get('counterName', '').startswith(counter_name):
                # Create a set of keys to track printed parameters
                printed_params = set()

                # Define the order of specific parameters to print first
                first_params = ['chipId', 'chipName', 'counterId', 'counterName', 'dropCount']
                
                # Print the first set of parameters
                for param in first_params:
                    if param in params:
                        print("{}: {}".format(param, params[param]))
                        printed_params.add(param)

                # Print the rest of the parameters
                for k, v in params.items():
                    if k not in printed_params:
                        print("{}: {}".format(k, v['value'] if isinstance(v, dict) else v))
                print("\n\n")

def main():
    counter_names = list(counter_id_mapping.values())
    parser = argparse.ArgumentParser(description='Process some integers.')
    parser.add_argument('--gnmi_arista', default='./gnmi_arista', help='Full Path to gnmi_arista executable')
    parser.add_argument('--addr', default='localhost:5910', help='Address')
    parser.add_argument('--username', default='admin', help='Username')
    parser.add_argument('--password', default='password', help='Password')
    parser.add_argument('--counterName', default='all', choices=counter_names + ['all'], help='Counter name, one of: ' + ', '.join(counter_names) + ', or all')
    parser.add_argument('--method', default='get', choices=['get', 'subscribe'], help='Method to use, either get or subscribe')
    args = parser.parse_args()

    if args.counterName not in counter_names and args.counterName != 'all':
        print("Error: Invalid counter name.")
        return

    if not os.path.isfile(args.gnmi_arista):
        print("Error: gnmi_arista not found.")
        return

    if not os.access(args.gnmi_arista, os.X_OK):
        print("Error: gnmi_arista is not executable.")
        return

    command = "{} -addr {} -username {} -password {}  -compression \"\" {} /Smash/hardware/counter/internalDrop/SandCounters/internalDrop".format(args.gnmi_arista, args.addr, args.username, args.password, args.method)
    print("Running command: {}".format(command))

    try:
        if args.method == 'get':
            output = run_command_get(command)
        else:
            output = run_command_subscribe(command)
    except FileNotFoundError:
        print("Error: gnmi_arista not found.")
        return
    except Exception as e:
        print("An error occurred while running the command: ", e)
        return

    if output is None:
        print("Failed to get output from command.")
        return

    parsed_data = parse_output(output)
    if not parsed_data:
        print("Parsed data is empty.")
        return

    print_table(parsed_data, args.counterName)

if __name__ == "__main__":
    main()