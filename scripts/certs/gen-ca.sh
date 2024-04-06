#!/bin/bash
set -ex
# generate CA's  key
openssl ecparam -name prime256v1 -genkey -noout -out ca.key

openssl req -config openssl.cnf -key ca.key -new -x509 -days 7300 -sha256 -extensions v3_ca -out ca.cert

#openssl s_server -key ca.key -cert ca.cert -accept 9999 -www