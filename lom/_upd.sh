#! /bin/bash

DST="/home/remanava/lom/remanava/lom"
UPD=""
MSG="Diff files ?"

if [ $# -eq 1 ]; then
    if [ "$1" = "upd" ]; then
        UPD="update"
        MSG="Update file ?"
        CMD="echo cp ${SRC_FL} ${DsT_FL}"
    fi
fi

if [ -z "${UPD}" ]; then
    echo "Compare only"
else
    echo "Do update"
    echo "Msg=${MSG}"
    echo "Run=${CMD}"
fi

libArray=(  "Makefile"
            "src/lib/lib_test/tx_test.go"
            "src/lib/lomcommon/config.go"
            "src/lib/lomcommon/hal.go"
            "src/lib/lomcommon/helper.go"
            "src/lib/lomipc/client_transport.go"
            "src/lib/lomipc/json_transport.go"
            "src/lib/lomipc/server_transport.go")

doDiff() {
    zdiff $1 $2 | less
}

doCp() {
    cp $1 $2
}

for i in ${libArray[@]}; do
    SRC_FL=$i
    DsT_FL=${DST}/$i

    echo -n "$i "
    if cmp --silent -- $i ${DST}/$i; then
        echo "================= identical ============="
    else
        echo ${MSG}
        select yn in "Yes" "No"; do
            case $yn in
                Yes )
                    if [ -z "${UPD}" ]; then
                        doDiff ${SRC_FL} ${DsT_FL}
                    else 
                        doCp ${SRC_FL} ${DsT_FL}
                    fi
                    break;;
                No ) echo "Skip"; break;;
            esac
        done
    fi
done
