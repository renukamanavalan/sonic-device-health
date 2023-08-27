#! /bin/bash

source $(dirname $0)/common.sh
set -x


function fStart()
{
    echo "************ function $@ START *******************"
}

function fEnd()
{
    echo "************ function $@ END   *******************"
}

function fail()
{
    # Mark it disabled on failure exit
    DBUpdate 0
    echo "---------------Terminating ...--------------------"
    echo "$1"
    exit $2
}

image_latest=0
image_backup=0
image_tag=""

function getTag()
{
    fStart getTag

    lst="$(docker images ${IMAGE_NAME} --format "{{.Tag}}")"
    read -r -a lstTags <<< "$(echo ${lst})"
    image_cnt="${#lstTags[@]}"
    if (( ${image_cnt} > 3 || ${image_cnt} == 1 )); then
        fail "Expect 0 or 2 or 3 tags. Current=${image_cnt}; Run clean" ${ERR_TEST}
    fi

    for tag in "${lstTags[@]}"
    do
        if [[ "${tag}" == "latest" ]]; then
            image_latest=1
        elif [[ "${tag}" == "${BACK_EXT}" ]]; then
            image_backup=1
        elif [[ "${image_tag}" != "" ]]; then
            fail "Duplicate image tag exist. tags=${lstTags[@]; Run clean}" ${ERR_TEST}
        else
            image_tag="${tag}"
        fi
    done
    if [[ ${image_cnt} != 0 ]]; then
        if [[ ${image_latest} == 0 || "${image_tag}" == "" ]]; then
            fail "Image latest or tag missing. tags=${lstTags[@]; Run clean}" ${ERR_TEST}
        fi
    fi
    echo "image_latest=${image_latest} image_backup=${image_backup} image_tag=${image_tag}"
    fEnd getTag
}


function filesTest()
{
    fStart filesTest Hostfiles $1
    present=0
    absent=0

    pushd /
    for i in ${HOST_FILES}; do
        if [[ ! -e "$i$1" ]]; then
            absent=1
        else
            present=1
        fi
    done
    popd

    if [[ ${present} == ${absent} ]]; then
        # Either all present or all absent is a valid state.
        # If not, reflects some corrupt state. Need cleaning.
        #
        fail "Partial Host files exist (${HOST_FILES}). Run cleanup"; ${ERR_TEST}
    fi
    fEnd filesTest Hostfiles $1
    return $present
}

function testInstall()
{
    fStart testInstall

    # Get image info & validate too.
    getTag

    filesTest
    hostfiles_exist=$?

    filesTest ".${BACK_EXT}"
    backup_file_exist=$?

    echo "hostfiles_exist=${hostfiles_exist} backup_file_exist=${backup_file_exist}"

    if [[ ${image_backup} != ${backup_file_exist} ]]; then
        fail "Partial backup. Image=${image_backup} files=${backup_file_exist}; Run clean" ${ERR_TEST}
    fi

    if [[ ${image_latest} != ${hostfiles_exist} ]]; then
        fail "Partial install. image=${image_latest} files=${hostfiles_exist}; Run clean" ${ERR_TEST}
    fi
    fEnd testInstall
}


function backUp()
{
    fStart backUp

    [[ ${image_backup} == 1 ]] && { fail "Backup pre-exists. Run clean;" ${ERR_BACKUP}; }
    [[ ${image_latest} == 0 ]] && { echo "Install don't exist. Nothing to backup;"; return 0; }

    pushd /
    for i in ${HOST_FILES}; do
        sudo mv $i $i.${BACK_EXT}
        [[ $? != 0 ]] && { fail "Failed to move $i to $i.${BACK_EXT}" ${ERR_BACKUP}; }
    done
    popd

    docker tag ${IMAGE_NAME}:latest ${IMAGE_NAME}:${BACK_EXT}
    [[ $? != 0 ]] && { fail "Failed to tag ${IMAGE_NAME} ${IMAGE_NAME}:${BACK_EXT}" ${ERR_BACKUP}; }

    docker rmi ${IMAGE_NAME}:latest
    [[ $? != 0 ]] && { fail "Failed to untag ${IMAGE_NAME}:latest" ${ERR_BACKUP}; }

    docker rmi ${IMAGE_NAME}:${image_tag}
    [[ $? != 0 ]] && { fail "Failed to untag ${IMAGE_NAME}:${image_tag}" ${ERR_BACKUP}; }

    fEnd backUp
}


function serviceStop()
{
    sudo systemctl stop device-health.service
    docker rm device-health
}


function DBUpdate()
{
    fStart DBUpdate

    if [[ $1 == 0 ]]; then
        state="disabled"
    elif [[ $1 == 1 ]]; then
        state="enabled"
    else
        fail "Internal error in usage. Expect arg as 1 or 0" ${ERR_DB}
    fi

    # Create FEATURE table entry
    RET="$(redis-cli -n 4 hmset "FEATURE|device-health" "auto_restart" "enabled" \
        "delayed" "True" "has_global_scope" "True" "has_per_asic_scope" "False" \
        "high_mem_alert" "disabled" "set_owner" "kube" "state" "${state}" \
        "support_syslog_rate_limit" "true")"

    [[ "${RET}" == "OK" ]] || { fail "failed to create/update FEATURE table entry for state=${state}" ${ERR_DB}; }

    fEnd DBUpdate
}


function forceClean() 
{   
    fStart forceClean
    sudo systemctl stop device-health.service 
    sudo systemctl disable device-health.service 
    docker rm device-health

    pushd / 
    for i in ${HOST_FILES}; do 
        sudo rm -f $i.${BACK_EXT}
        [[ $? != 0 ]] && { fail "Failed to remove $i.${BACK_EXT}" ${ERR_CLEAN}; }
        sudo rm -f $i
        [[ $? != 0 ]] && { fail "Failed to remove $i" ${ERR_CLEAN}; }
    done
    popd

    lst="$(docker images ${IMAGE_NAME} --format "{{.Tag}}")"
    read -r -a lstTags <<< "$(echo ${lst})"
    for tag in "${lstTags[@]}"
    do
        docker rmi ${IMAGE_NAME}:${tag}
        [[ $? != 0 ]] && { fail "Failed to untag ${IMAGE_NAME}:${tag}" ${ERR_CLEAN}; }
    done
    DBUpdate 0
    fEnd forceClean
}


function installCode()
{
    fStart installCode
    pushd $(dirname $0)/../host

    for i in ${HOST_FILES}; do
        sudo cp $i /$i
    done
    popd

    fl="$(dirname $0)/../install/${IMAGE_FILE}"
    docker load -i ${fl}
    [[ $? != 0 ]] && { fail "Failed to load docker image ${fl}" ${ERR_INSTALL_CODE}; }
    tag="$(cat $(dirname $0)/VERSION |  tr -d '\n')"
    docker tag ${IMAGE_NAME}:latest ${IMAGE_NAME}:${tag}
    [[ $? != 0 ]] && { fail "Failed to tag ${IMAGE_NAME}:latest to ${tag}" ${ERR_INSTALL_CODE}; }

    fEnd installCode
}


function rollBackCode()
{
    [[ ${image_backup} == 0 ]] && { fail "Backup don't exist. Nothing to rollback;" ${ERR_ROLLBACK}; }
    [[ ${image_latest} == 1 ]] && { fail "Install exists. Run cleanup;" ${ERR_ROLLBACK}; }

    fStart rollBackCode

    # Check all backup files exist
    pushd /
    for i in ${HOST_FILES}; do
        [[ ! -e $i.${BACK_EXT} ]] && { fail "Backup file $i.${BACK_EXT} does not exist" ${ERR_ROLLBACK}; }
    done
    popd

    # Check backup image exists.
    if [[ "$(docker images -q ${IMAGE_NAME}:${BACK_EXT} 2> /dev/null)" == "" ]]; then
        fail "Docker image backup do not exist ${IMAGE_NAME}:${BACK_EXT}." ${ERR_ROLLBACK}
    fi

    pushd /
    for i in ${HOST_FILES}; do
        sudo rm -f $i
        sudo mv $i.${BACK_EXT} $i
        [[ $? != 0 ]] && { fail "Failed to rollback  mv $i.${BACK_EXT} $i;" ${ERR_ROLLBACK}; }
    done
    popd

    if [[ "$(docker images -q ${IMAGE_NAME}:latest 2> /dev/null)" != "" ]]; then
        docker rmi ${IMAGE_NAME}:latest
        [[ $? != 0 ]] && { fail "Failed to untag ${IMAGE_NAME}:latest" ${ERR_ROLLBACK}; }
    fi

    docker tag ${IMAGE_NAME}:${BACK_EXT} ${IMAGE_NAME}:latest
    [[ $? != 0 ]] && { fail "Failed to tag ${IMAGE_NAME}:${BACK_EXT} ${IMAGE_NAME}:latest" ${ERR_ROLLBACK}; }

    docker rmi ${IMAGE_NAME}:${BACK_EXT}
    [[ $? != 0 ]] && { fail "Failed to untag ${IMAGE_NAME}:${BACK_EXT}" ${ERR_ROLLBACK}; }

    fEnd rollBackCode
}


function serviceRestart()
{
    fStart serviceRestart

    sudo systemctl daemon-reload
    sudo systemctl enable device-health.service 
    sudo systemctl reset-failed device-health.service 

    # Create/Update FEATURE table entry.
    #
    DBUpdate 1

    # Restart installed/upgraded/rolledback service instance.
    #
    sudo systemctl restart device-health.service 
    [[ $? != 0 ]] && { fail "Failed to restart device-health service" ${ERR_RESTART}; }

    # Pause for a minute
    #
    sleep 1m

    # Check running state of critical processes.
    #
    for i in LoMEngine LoMPluginMgr; do
        pidof $i
        [[ $? != 0 ]] && { fail "LoM Process $i is not running" ${ERR_RUNTIME}; }
    done

    # Have a custom post-check script inside LoM container
    #
    # docker exec -it device-health /usr/bin/post-install-check.sh
    # [[ $? != 0 ]] && { fail "LoM post-install-check failed" ${ERR_RUNTIME}; }
    #

    fEnd serviceRestart
}


function usage()
{
    echo -e "\
        -i - Does install or upgrade \n\
        -f - Force a clean up of backup for upgrade if needed \n\
        -r - Force a rollback \n\
        -c - Clean all the backup \n\
        -h - Usage"
    exit ${ERR_USAGE}
}


function main()
{
    OP_INSTALL=0        # Install or upgrade. Upgrade fails if backup exists unless forced.
    OP_CLEAN=0          # Clean any backup  
    OP_ROLLBACK=0       # Rollback to last backup

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

    # Clean stop service first
    serviceStop
    
    if [[ ${OP_CLEAN} == 1 ]]; then
        forceClean
    fi

    testInstall

    if [[ ${OP_ROLLBACK} == 1 ]]; then
        rollBackCode
        serviceRestart
    elif [[ ${OP_INSTALL} == 1 ]]; then
        backUp
        installCode
        serviceRestart
    fi
    echo "\"$0 $@\" - Ran successfully"
    exit 0
}

main $@

