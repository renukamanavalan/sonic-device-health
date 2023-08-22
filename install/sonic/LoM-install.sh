#! /bin/bash


source $(dirname $0)/common.sh


function testInstall()
{
    if [[ "$(docker images -q ${IMAGE_NAME}:latest 2> /dev/null)" != "" ]]; then
        echo "running"
    else 
        echo ""
    fi
}


function backUp()
{
    # Make sure there are no pre-existing backups; Fail if any.
    # Pre-existing implies that last run is botched.
    # Do an assessment followed by deep clean before upgrade.
    # Any upgrade takes a back up at the start and clean it up upon successful install/upgrade.
    #
    pushd /
    for i in ${HOST_FILES}; do
        [[ ! -e $i ]] && { echo "File to backup $i does not exist"; exit ${ERR_BACKUP}; }
        [[ -e $i.${BACK_EXT} ]] && { echo "Backup file $i.${BACK_EXT} exists; clear it"; exit ${ERR_BACKUP}; }
    done
    popd

    if [[ "$(docker images -q ${IMAGE_NAME}:latest 2> /dev/null)" == "" ]]; then
        echo "Docker image to backup does not exist ${IMAGE_NAME}:latest."
        exit ${ERR_BACKUP}
    fi

    if [[ "$(docker images -q ${IMAGE_NAME}:$[BACK_EXT} 2> /dev/null)" != "" ]]; then
        echo "Docker image backup pre-exists ${IMAGE_NAME}:$[BACK_EXT}. Clear it"
        exit ${ERR_BACKUP}
    fi

    for i in ${HOST_FILES}; do
        mv $i $i.${BACK_EXT}
        [[ $? != 0 ]] && { echo "Failed to move $i to $i.${BACK_EXT}"; exit ERR_BACKUP; }
    done
    
    docker tag ${IMAGE_NAME}:latest ${IMAGE_NAME}:${BACK_EXT}
    [[ $? != 0 ]] && { echo "Failed to tag ${IMAGE_NAME} ${IMAGE_NAME}:${BACK_EXT}"; exit ${ERR_BACKUP}; }

    docker rmi ${IMAGE_NAME}:latest
    [[ $? != 0 ]] && { echo "Failed to untag ${IMAGE_NAME}:latest"; exit ${ERR_BACKUP}; }

    echo "Function backup complete"
}


function forceClean() 
{   
    pushd / 
    for i in ${HOST_FILES}; do 
        rm -f $i.${BACK_EXT}
        [[ $? != 0 ]] && { echo "Failed to remove $i.${BACK_EXT}; exit ${ERR_CLEAN}; }
    done
    popd

    if [[ "$(docker images -q ${IMAGE_NAME}:$[BACK_EXT} 2> /dev/null)" != "" ]]; then
        docker rmi ${IMAGE_NAME}:$[BACK_EXT}
        [[ $? != 0 ]] && { echo "Failed to untag ${IMAGE_NAME}:$[BACK_EXT}"; exit ${ERR_CLEAN}; }
    fi
    echo "Function forceClean complete"
}


function installCode()
{
    pushd $(dirname $0)/../host

    for i in ${HOST_FILES}; do
        cpFile $i /$i
    done
    popd

    fl="$(dirname $0)/../install/${IMAGE_FILE}"
    docker load ${fl}
    [[ $? != 0 ]] && { echo "Failed to load docker image ${fl}"; exit ${ERR_INSTALL_CODE}; }
    echo "Function installCode complete"
}


function DBUpdate()
{
    # Create FEATURE table entry
    RET=$(redis-cli -n 4 hmset "FEATURE|device-health" "auto_restart" "enabled" \
        "delayed" "True" "has_global_scope" "True" "has_per_asic_scope" "False" \
        "high_mem_alert" "disabled" "set_owner" "kube" "state" "enabled" \
        "support_syslog_rate_limit" "true")

    [[ "${RET]" == "OK" ]] || { echo "failed to create FEATURE table entry"; exit -1; }
    echo "Function DBUpdate complete"
}


function rollBackCode()
{
    # Check all backup files exist
    pushd /
    for i in ${HOST_FILES}; do
        [[ ! -e $i.${BACK_EXT} ]] && { echo "Backup file $i.${BACK_EXT} does not exist"; exit ${ERR_ROLLBACK}; }
    done
    popd

    # Check backup image exists.
    if [[ "$(docker images -q ${IMAGE_NAME}:$[BACK_EXT} 2> /dev/null)" == "" ]]; then
        echo "Docker image backup do not exist ${IMAGE_NAME}:$[BACK_EXT}."
        exit ${ERR_ROLLBACK}
    fi

    pushd /
    for i in ${HOST_FILES}; do
        rm -f $i
        mv $i.${BACK_EXT} $i
        [[ $? != 0 ]] && { echo "Failed to rollback  mv $i.${BACK_EXT} $i;"; exit ${ERR_ROLLBACK}; }
    done
    popd

    if [[ "$(docker images -q ${IMAGE_NAME}:latest 2> /dev/null)" != "" ]]; then
        docker rmi ${IMAGE_NAME}:latest
        [[ $? != 0 ]] && { echo "Failed to untag ${IMAGE_NAME}:latest"; exit ${ERR_ROLLBACK}; }
    fi

    docker tag ${IMAGE_NAME}:$[BACK_EXT} ${IMAGE_NAME}:latest
    [[ $? != 0 ]] && { echo "Failed to tag ${IMAGE_NAME}:$[BACK_EXT} ${IMAGE_NAME}:latest"; exit ${ERR_ROLLBACK}; }

    docker rmi ${IMAGE_NAME}:$[BACK_EXT}
    [[ $? != 0 ]] && { echo "Failed to untag ${IMAGE_NAME}:$[BACK_EXT}"; exit ${ERR_ROLLBACK}; }

    echo "Function rollbackCode complete"
}


function serviceRestart()
{
    # Create/Update FEATURE table entry.
    #
    DBUpdate

    # Restart installed/upgraded/rolledback service instance.
    #
    sudo systemctl restart device-health.service 
    [[ $? != 0 ]] && { echo "Failed to restart device-health service"; exit ${ERR_ROLLBACK}; }

    # Pause for a minute
    #
    sleep 1m

    # Check running state of critical processes.
    #
    for i in LoMEngine LoMPluginMgr; do
        pidof $i
        [[ $? != 0 ]] && { echo "LoM Process $i is not running"; exit ${ERR_RUNTIME}; }
    done

    # Have a custom post-check script inside LoM container
    #
    # docker exec -it device-health /usr/bin/post-install-check.sh
    # [ $? != 0 ]] && { echo "LoM post-install-check failed"; exit ${ERR_RUNTIME}; }
    #
    echo "LoM Service restarted successfully"
}


function usage()
{
    echo "\
        -i - Does install or upgrade \
        -f - Force a clean up of backup for upgrade if needed \
        -r - Force a rollback \
        -c - Clean all the backup \
        -h - Usage"
    exit ${ERR_USAGE}
}


function main()
{
    OP_INSTALL=0        # Install or upgrade. Upgrade fails if backup exists unless forced.
    OP_CLEAN=0          # Clean any backup  
    OP_ROLLBACK=0       # Rollback to last backup

    op=OP_NONE

    while getopts "hifcr" opt; do
        case ${opt} in
            i ) OP_INSTALL=1
                ;;
            r ) OP_ROLLBACK=1
                ;;
            c ) OP_CLEAN=1
                ;;
            h ) usage
                ;;
            \? ) usage
                ;;
        esac
    done

    
    if [[ OP_CLEAN == 1 ]]; then
        forceClean
    fi

    if [[ OP_ROLLBACK == 1 ]]; then
        rollBackCode
        serviceRestart
    fi

    if [[ OP_INSTALL == 1 ]]; then
        installCode
        serviceRestart
    fi
    echo "\"$0 $@\" - Ran successfully"
    exit 0
}

main $@
