#!/bin/bash
set -ex

sudo cp ca.cert /usr/local/share/ca-certificates/jupyter-proxy.crt
sudo update-ca-certificates
