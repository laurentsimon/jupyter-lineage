#!/bin/bash
set -ex

sudo rm /usr/local/share/ca-certificates/jupyter-proxy.crt
sudo sudo update-ca-certificates --fresh