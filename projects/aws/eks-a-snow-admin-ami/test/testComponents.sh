#!/bin/bash

INSTANCE_TYPE=${INSTANCE_TYPE:-'t2.large'}
AMI_ID=${AMI_ID:-'ami-0baebd2c53afce272'}
KEY_NAME=$1
KEY_PATH=$2
USER=ubuntu
DOCUMENTS=components/install_eksa.yaml,components/download_eksa_artifacts.yaml
PHASES=build,validate
PARAMETERS=$3

./test/toeInInstance.sh $AMI_ID $INSTANCE_TYPE $KEY_NAME $KEY_PATH $USER $DOCUMENTS $PHASES $PARAMETERS
