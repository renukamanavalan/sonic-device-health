#! /bin/bash

DST="/home/remanava/lom/remanava/lom"

libArray=(  "Makefile"
            "src/lib/README.md"
            "src/lib/clib/lom_clib.go"
            "src/lib/lib_test/tx_test.go"
            "src/lib/lomcommon/config.go"
            "src/lib/lomcommon/hal.go"
            "src/lib/lomcommon/helper.go"
            "src/lib/lomipc/client_transport.go"
            "src/lib/lomipc/json_transport.go"
            "src/lib/lomipc/server_transport.go"
            "src/engine/README.md"
            "src/engine/context.go"
            "src/engine/engine.go"
            "src/engine/engine_data_test.go"
            "src/engine/engine_test.go"
            "src/engine/engine_ut_test.go"
            "src/engine/sequenceHandler.go"
            "src/engine/serverReqHandler.go"
            "python/README.rst"
            "python/pytest.ini"
            "python/setup.cfg"
            "python/setup.py"
            "python/src/__init__.py"
            "python/src/common/common.py"
            "python/src/common/engine_apis.py"
            "python/src/common/engine_rpc_if.py"
            "python/src/common/gvars.py"
            "python/tests/README"
            "python/tests/__init__.py"
            "python/tests/common_test.py"
            "python/tests/engine_apis_test.py"
        )

CMD="doDiff"
UPD=0
if [ $# -ne 0 ] && [ "$1" == "upd" ]
then
    UPD=1
    CMD="doCp"
fi

doDiff() {
    zdiff $1 $2 | less
}

doCp() {
    if ! test -d $(dirname ${DST}/$i); then
        mkdir -p $(dirname ${DST}/$i)
    fi
    cp $1 $2
}

do_check_upd() {
    if [ $UPD -eq 0 ]; then
        echo "Check $1"
    else
        echo "Update $1"
    fi


    select ynx in "Yes" "No" "Exit"; do
        echo "($ynx)"
        case $ynx in
            Yes ) ${CMD} ${SRC_FL} ${DsT_FL}; break;;
            No ) echo "Skip"; break;;
            Exit ) echo "Exiting ..."; exit;;
        esac
    done
}


for i in ${libArray[@]}; do
    SRC_FL=$i
    DsT_FL=${DST}/$i

    if cmp --silent -- $i ${DST}/$i; then
        echo "================= $i identical ============="
    elif ! test -f ${DST}/$i; then
        echo "================= $i  New =============="
        if [ $UPD -ne 0 ]; then
            do_check_upd ${SRC_FL} ${DsT_FL}
        fi
    else
        do_check_upd ${SRC_FL} ${DsT_FL}
    fi
done

