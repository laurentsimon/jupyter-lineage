#!/bin/bash
set -ex
# Generate CA's  key
openssl ecparam -name secp521r1 -genkey -noout -out ca.key
# Generate the CA cert.
openssl req -batch -config openssl.cnf -key ca.key -new -x509 -days 7300 -sha256 -extensions v3_ca -out ca.cert
