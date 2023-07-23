import subprocess
import sys

import api

# Function to kill LoMEngine
def kill_lomengine():
    engine_pids = api.get_lomengine_pids()
    if engine_pids:
        print("Killing LoMEngine instances...")
        for pid in engine_pids:
            print(f"Killing LoMEngine (PID: {pid})...")
            subprocess.run(['sudo', "kill", str(pid)])
        print("LoMEngine instances killed.")
    else:
        print("No LoMEngine instances are running.")

# Function to kill all LoMPluginMgr instances
def kill_lompluginmgr():
    pluginmgr_pids = api.get_lompluginmgr_pids()
    if pluginmgr_pids:
        print("Killing LoMPluginMgr instances...")
        for pid in pluginmgr_pids:
            print(f"Killing LoMPluginMgr (PID: {pid})...")
            subprocess.run(['sudo', "kill", str(pid)])
        print("LoMPluginMgr instances killed.")
    else:
        print("No LoMPluginMgr instances are running.")

# Print usage information
def print_usage():
    print("Usage: python kill.py [all|engine|plugin]")
    print("  all    : Kill both LoMEngine and LoMPluginMgr")
    print("  engine : Kill LoMEngine")
    print("  plugin : Kill all LoMPluginMgr instances")

if __name__ == '__main__':
    # Check the argument
    if len(sys.argv) != 2:
        print_usage()
        exit(1)

    arg = sys.argv[1]
    if arg == "all":
        kill_lomengine()
        kill_lompluginmgr()
    elif arg == "engine":
        kill_lomengine()
    elif arg == "plugin":
        kill_lompluginmgr()
    else:
        print_usage()
        exit(1)
