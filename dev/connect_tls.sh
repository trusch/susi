#!/bin/bash

PORT=$1

gnutls-cli -p $PORT --insecure --x509certfile "dev/cert.pem" --x509keyfile "dev/key.pem" localhost

exit $?
