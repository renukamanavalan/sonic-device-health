#! /bin/bash

source $(dirname $0)/common.sh

# Enable set -x when you need to debug ...
# set -x

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
image_tag=""
image_backup=0
image_backup_tag=""

function backup_tag()
{
    # Backup tag is coined as "${BACK_EXT}_{original image tag}
    # If called with ${BACK_EXT}_, it strips and saves the original tag as backup_Tag
    # Elif called w/o ${BACK_EXT}_, it creates full tag with ${BACK_EXT}_ prefixed
    # Return code distinguishes the 2 scenarios
    #
    if [[ "$1" =~ ^${BACK_EXT}.* ]]; then
        backupTag="$(echo $1 | cut -d'_' -f2-)"
        return 1
    else
        backupTag="${BACK_EXT}_$1"
        return 0
    fi
}


function getTag()
{
    # Get image state & tags.
    # Fail on inconsistent state
    #
    fStart getTag

    image_latest=0
    image_tag=""
    image_backup=0
    image_backup_tag=""

    lst="$(docker images ${IMAGE_NAME} --format "{{.Tag}}")"
    read -r -a lstTags <<< "$(echo ${lst})"
    image_cnt="${#lstTags[@]}"
    if (( ${image_cnt} > 3 || ${image_cnt} == 1 )); then
        fail "Expect 0 or 2 or 3 tags. Current=${image_cnt}; Run clean" ${ERR_TEST}
    fi

    for tag in "${lstTags[@]}"
    do
        backup_tag ${tag}
        is_backup=$?

        if [[ "${tag}" == "latest" ]]; then
            image_latest=1
        elif [[ ${is_backup} == 1 ]]; then
            [[ ${image_backup} == 1 ]] && fail "Duplicate backup image exist. tags=${lstTags[@]}; Run clean" ${ERR_TEST}
            image_backup_tag="${backupTag}"
            image_backup=1
        elif [[ "${image_tag}" != "" ]]; then
            fail "Duplicate image tag exist. tags=${lstTags[@]}; Run clean" ${ERR_TEST}
        else
            image_tag="${tag}"
        fi
    done
    if [[ ${image_cnt} != 0 ]]; then
        if [[ ${image_latest} == 0 || "${image_tag}" == "" ]]; then
            lst=$(echo "$(IFS=,; echo "${lstTags[*]}")")
            fail "Image latest or tag missing. tags are { ${lst} }; Run clean." ${ERR_TEST}
        fi
    fi
    echo "image_latest=${image_latest} image_backup=${image_backup}"
    echo "image_tag=${image_tag} image_backup_tag=${image_backup_tag}"
    fEnd getTag
}


function filesTest()
{
    # Test presence of files
    #
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
    # Test install state consistency
    # Fail if corrupt
    #
    fStart testInstall

    # Match the build from LoM's version with switch's version
    # Take first component from install/VERSION string
    # Take second component from /etc/sonic/sonic_version.yml's build_version's value
    #
    # TODO: This breaks for private builds. Need to mature before bringing this constraint.
    #
    # OS_Version=$(cat /etc/sonic/sonic_version.yml | grep -e "^build_version" | cut -f2 -d\'| cut -f1 -d .)
    # LoM_Version="$(cat $(dirname $0)/VERSION |  tr -d '\n' | cut -f1 -d .)"
    # [[ ${OS_Version} != ${LoM_Version} ]] && fail "Version mismatch. OS=${OS_Version} LoM=${LoM_Version}" ${ERR_TEST}

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


function forceClean() 
{   
    # $1 == 1 clean install only
    # $1 == 2 clean backup only
    # $1 == 3 clean all
    #
    [[ $1 < 1 || $1 > 3 ]] && fail "Internal usage error" ${ERR_CLEAN}

    fStart forceClean $@
    sudo systemctl stop device-health.service 
    sudo systemctl disable device-health.service 
    docker rm device-health

    bClean=$(( $1 & 2 ))
    iClean=$(( $1 & 1 ))

    sudo rm -rf /usr/share/device_health/*
    pushd / 
    for i in ${HOST_FILES}; do 
        if [[ ${bClean} != 0 ]]; then
            sudo rm -f $i.${BACK_EXT}
            [[ $? != 0 ]] && { fail "Failed to remove $i.${BACK_EXT}" ${ERR_CLEAN}; }
        fi
        if [[ ${iClean} != 0 ]]; then
            sudo rm -f $i
            [[ $? != 0 ]] && { fail "Failed to remove $i" ${ERR_CLEAN}; }
        fi
    done
    popd

    # truth table
    # If $1 == 3 delete any
    # iClean bClean is_backup delete
    # 0      1      0         No
    # 1      0      0         Yes
    # 0      1      1         Yes
    # 1      0      1         No
    # if $bclean == $is_backup remove
    # 
    lst="$(docker images ${IMAGE_NAME} --format "{{.Tag}}")"
    read -r -a lstTags <<< "$(echo ${lst})"
    for tag in "${lstTags[@]}"
    do
        backup_tag "${tag}"
        is_backup=$?

        if [[ $1 == 3 || ${is_backup} == ${bClean} ]]; then
            docker rmi ${IMAGE_NAME}:${tag}
            [[ $? != 0 ]] && { fail "Failed to untag ${IMAGE_NAME}:${tag}" ${ERR_CLEAN}; }
        fi
    done
    DBUpdate 0
    fEnd forceClean $@
}

function backUp()
{
    # Back up current install
    # If any backup pre-exists, remove it.
    #
    fStart backUp

    [[ ${image_latest} == 0 ]] && { echo "Install don't exist. Nothing to backup;"; return 0; }

    # Remove existing backup
    if [[ ${image_backup} == 1 ]]; then
        forceClean 2
    fi

    pushd /
    for i in ${HOST_FILES}; do
        sudo mv $i $i.${BACK_EXT}
        [[ $? != 0 ]] && { fail "Failed to move $i to $i.${BACK_EXT}" ${ERR_BACKUP}; }
    done
    popd

    # coins backup tag with BACK_EXT
    backup_tag "${image_tag}"
    docker tag ${IMAGE_NAME}:latest ${IMAGE_NAME}:${backupTag}
    [[ $? != 0 ]] && { fail "Failed to tag ${IMAGE_NAME} ${IMAGE_NAME}:${BACK_EXT}" ${ERR_BACKUP}; }

    docker rmi ${IMAGE_NAME}:latest
    [[ $? != 0 ]] && { fail "Failed to untag ${IMAGE_NAME}:latest" ${ERR_BACKUP}; }

    docker rmi ${IMAGE_NAME}:${image_tag}
    [[ $? != 0 ]] && { fail "Failed to untag ${IMAGE_NAME}:${image_tag}" ${ERR_BACKUP}; }

    fEnd backUp
}


function serviceStop()
{
    fStart serviceStop
    sudo systemctl daemon-reload
    sudo systemctl stop device-health.service
    docker rm device-health
    fEnd serviceStop
}


function DBUpdate()
{
    fStart DBUpdate

    if [[ $1 == 0 ]]; then
        redis-cli -n 4 del "FEATURE|device-health"
    elif [[ $1 == 1 ]]; then
        # Create FEATURE table entry
        RET="$(redis-cli -n 4 hmset "FEATURE|device-health" "auto_restart" "enabled" \
            "delayed" "True" "has_global_scope" "True" "has_per_asic_scope" "False" \
            "high_mem_alert" "disabled" "set_owner" "kube" "state" "enabled" \
            "support_syslog_rate_limit" "true")"

        [[ "${RET}" == "OK" ]] || { fail "failed to create FEATURE table entry" ${ERR_DB}; }
    else
        fail "Internal error in usage. Expect arg as 1 or 0" ${ERR_DB}
    fi


    fEnd DBUpdate
}



function installCode()
{
    fStart installCode
    pushd $(dirname $0)/../host

    for i in ${HOST_FILES}; do
        sudo cp $i /$i
    done
    popd

    sudo mkdir -p /usr/share/device_health/
    [[ $? != 0 ]] && { fail "Failed to create /usr/share/lom/ dir" ${ERR_INSTALL_CODE}; }

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
    # Remove current install
    forceClean 1

    # Rollback to last back up version
    # Remove current install, if any
    # Rename back up files as current and backup image as latest
    #
    [[ ${image_backup} == 0 ]] && { echo "Backup don't exist. Nothing to rollback"; return; }

    fStart rollBackCode

    pushd /
    for i in ${HOST_FILES}; do
        sudo mv $i.${BACK_EXT} $i
        [[ $? != 0 ]] && { fail "Failed to rollback  mv $i.${BACK_EXT} $i;" ${ERR_ROLLBACK}; }
    done
    popd

    # coins backup tag with BACK_EXT 
    backup_tag "${image_backup_tag}"
    docker tag ${IMAGE_NAME}:${backupTag} ${IMAGE_NAME}:latest
    [[ $? != 0 ]] && { fail "Failed to tag ${IMAGE_NAME}:${BACK_EXT} ${IMAGE_NAME}:latest" ${ERR_ROLLBACK}; }

    docker tag ${IMAGE_NAME}:latest ${IMAGE_NAME}:${image_backup_tag}
    [[ $? != 0 ]] && { fail "Failed to tag ${IMAGE_NAME}:${BACK_EXT} ${IMAGE_NAME}:latest" ${ERR_ROLLBACK}; }

    docker rmi ${IMAGE_NAME}:${backupTag}
    [[ $? != 0 ]] && { fail "Failed to untag ${IMAGE_NAME}:${backupTag}" ${ERR_ROLLBACK}; }

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
    echo "Take a 1 min pause to settle down"
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

function testSudo()
{
    sudo -n true
    ret=$?
    if [[ ${ret} != 0 ]]; then
        fail "Require sudo privileges to run install" ${ERR_USAGE}
    fi
}


function main()
{
    testSudo

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
        # Clean all ( current install & backup )
        forceClean 3
    fi

    testInstall

    if [[ ${OP_ROLLBACK} == 1 ]]; then
        rollBackCode
    elif [[ ${OP_INSTALL} == 1 ]]; then
        backUp
        installCode
    fi
    testInstall
    if [[ ${image_latest} == 1 ]]; then
        serviceRestart
    fi
    echo "\"$0 $@\" - Ran successfully"
    exit 0
}

main $@

