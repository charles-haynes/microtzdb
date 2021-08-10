#! /bin/sh

set -xe

mkdir -p tzdir
cd tzdir
curl -L https://www.iana.org/time-zones/repository/tzcode-latest.tar.gz | tar -xzv
curl -L https://www.iana.org/time-zones/repository/tzdata-latest.tar.gz | tar -xzv

make TOPDIR=. install
cd ..
go build
printf "// built from tzdb version $(cat tzdir/version)\n\n#include <string.h>\n\n" >../microtzdb.c
./microtzdb tzdir/usr/share/zoneinfo >>../microtzdb.c
cat footer >>../microtzdb.c