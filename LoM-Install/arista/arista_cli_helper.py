"""
arista_cli_helper.py

This script provides a helper class for interacting with an Arista switch via the CLI.
"""

from __future__ import print_function  # Python 2/3 compatibility
import subprocess
import time
import argparse

from common import *

CONFGURATION_WAIT_TIME = 3  # Seconds to wait for the configuration to take effect

class AristaSwitchCLIHelper(object):
    def __init__(self):
        pass

    def _execute_arista_command(self, command, option='show', privilege_level=15, print_output=False):
        try:
            if option == 'show':
                full_command = 'Cli -c "{}"'.format(command)
            elif option == 'config':
                full_command = 'Cli -p {} -c "{}"'.format(privilege_level, command)
            else:
                raise ValueError("Invalid option. Use 'show' or 'config'.")
            output = subprocess.check_output(
                full_command, shell=True, stderr=subprocess.STDOUT, universal_newlines=True
            )            
            print("Command executed: {}".format(full_command))            
            if print_output:
                print_with_separator(output)            
            return output, None  # Return the output and indicate no exception
        except subprocess.CalledProcessError as e:
            return None, e.output.strip()  # Return None for output and the exception message
        except Exception as e:
            return None, str(e)  # Return None for output and a generic exception message

    def _is_unix_eapi_running(self):
        arista_command = 'show management api http-commands'
        output, error = self._execute_arista_command(arista_command, option='show', print_output=True)
        if error:
            return False, error
        return 'Unix Socket server: running' in output, None

    def _enable_unix_eAPI_protocol(self):
        arista_command = "configure\n\
                          management api http-commands\n\
                          protocol unix-socket\n\
                          no shutdown"
        output, error = self._execute_arista_command(arista_command, option='config', print_output=True)    
        if error:
            return False, error  
        time.sleep(CONFGURATION_WAIT_TIME)  # Wait for the configuration to take effect
        return self._is_unix_eapi_running(), None  # Return the result and no error

    def check_and_enable_unix_eAPI_protocol(self):
        running, error = self._is_unix_eapi_running()
        already_enabled = running
        if not running and not error:
            # Unix eAPI Socket is not running, try to enable it
            running, error = self._enable_unix_eAPI_protocol()
        return running, already_enabled, error
        

def main():
    parser = argparse.ArgumentParser(description='Arista Switch CLI Helper')
    parser.add_argument('--api', choices=['is_unix_eapi_running', 'enable_unix_eAPI_protocol', 'check_and_enable_unix_eAPI_protocol'], default='check_and_enable_unix_eAPI_protocol',
                        help='API to run. Choices are: is_unix_eapi_running, enable_unix_eAPI_protocol, check_and_enable_unix_eAPI_protocol')
    args = parser.parse_args()

    arista_manager = AristaSwitchCLIHelper()

    if args.api == 'is_unix_eapi_running':
        success, result = arista_manager._is_unix_eapi_running()
    elif args.api == 'enable_unix_eAPI_protocol':
        success, result = arista_manager._enable_unix_eAPI_protocol()
    elif args.api == 'check_and_enable_unix_eAPI_protocol':
        success, result = arista_manager.check_and_enable_unix_eAPI_protocol()

    if success:
        print("Success: {}".format(result))
    else:
        print("Error: {}".format(result))

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print("An error occurred in the AristaSwitchCLIHelper: {}".format(e))