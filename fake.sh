#! /bin/bash
IFS=$'\n'
for line in `cat lines`; do echo $line; sleep 2; done
