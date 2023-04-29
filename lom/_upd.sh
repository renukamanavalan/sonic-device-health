#! /bin/bash

DST="/home/remanava/lom/remanava/lom"

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

    if cmp --silent -- $i ${DST}/$i; then
        echo "================= $i identical ============="
    else
        echo "Check $i ?"
        select ynx in "Yes" "No" "Exit"; do
            case $ynx in
                Yes )
                    doDiff ${SRC_FL} ${DsT_FL}
                    echo "***** Update $i?"
                    select yn in "Yes" "No"; do
                        case $yn in
                            Yes ) doCp ${SRC_FL} ${DsT_FL}; break;;
                            No ) echo "No update"; break;;
                        esac
                    done
                    break;;
                No ) echo "Skip"; break;;
                Exit ) echo "Exiting ..."; exit;;
            esac
        done
    fi
done

