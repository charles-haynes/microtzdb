#! /bin/sh

mkdir -f tzdir
cd tzdir
curl -L https://www.iana.org/time-zones/repository/tzcode-latest.tar.gz | tar -xzv
curl -L https://www.iana.org/time-zones/repository/tzdata-latest.tar.gz | tar -xzv
