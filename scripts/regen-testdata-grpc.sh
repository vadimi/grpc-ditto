#!/bin/bash
cwd=$(dirname "$0")
bash $cwd/regen-grpc.sh testdata/greet
