#! /bin/bash

find ./ -type f | grep -v -e "^...git\|^..test\|^..patches\|^..doc\^..debian"
