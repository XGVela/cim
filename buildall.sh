#!/bin/bash
# Copyright 2020 Mavenir
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

##############################################
#usage: ./buildall.sh <CIM_VERSION> [<CLEAN_BUILD_BUILDER_IMAGE>] [<DELETE_BUILDER_IMAGE>]
# CIM_VERSION :(Mandatory): This argument can be passed from jenkins job or manual, Artifact cim container image will be tagged with this version.
# CLEAN_BUILD_BUILDER_IMAGE:(Optional): ["yes"|"no"] (Default: no). This will clean do clean build builder image.
# DELETE_BUILDER_IMAGE:(Optional): ["yes"|"no"] (Default: no). This will delete the image after artifacts are released.
##############################################

set -e

BASE_DISTRO_IMAGE="alpine"
BASE_DISTRO_VERSION="3.11.3"

BUILDER_IMAGE="cim-builder"
BUILDER_VERSION="v0.1"

MICROSERVICE_NAME="cim"
MICROSERVICE_VERSION=$1

CLEAN_BUILD_BUILDER_IMAGE=$2
DELETE_BUILDER_IMAGE=$3

ARTIFACTS_PATH="./artifacts"
BUILDER_ARG=""
COMMON_ARG=""
CONFIG_RELEASE_LIST="cim.yang cim.json eventdef-cim.json"

if [[ -n "$CLEAN_BUILD_BUILDER_IMAGE" ]] && [[ "$CLEAN_BUILD_BUILDER_IMAGE" == "yes" ]]; then
echo -e "\e[1;32;40m[CIM-BUILD] Clean Build Builder Image...\e[0m"
BUILDER_ARG="--no-cache"
fi

echo -e "\e[1;32;40m[CIM-BUILD] Build Builder:$BUILDER_IMAGE, Version:$BUILDER_VERSION \e[0m"
docker build --rm $BUILDER_ARG \
             $COMMON_ARG \
             --build-arg BASE_DISTRO_IMAGE=$BASE_DISTRO_IMAGE \
             --build-arg BASE_DISTRO_VERSION=$BASE_DISTRO_VERSION \
             -f ./build_spec/cim-builder_dockerfile \
             -t $BUILDER_IMAGE:$BUILDER_VERSION .

##NANO SEC timestamp LABEL, to enable multiple build in same system
BUILDER_LABEL="cim-builder-$(date +%s%9N)"
echo -e "\e[1;32;40m[CIM-BUILD] Build MICROSERVICE_NAME:$MICROSERVICE_NAME, Version:$MICROSERVICE_VERSION \e[0m"
docker build --rm \
             $COMMON_ARG \
             --build-arg BUILDER_LABEL=$BUILDER_LABEL \
             --build-arg BUILDER_IMAGE=$BUILDER_IMAGE \
             --build-arg BUILDER_VERSION=$BUILDER_VERSION \
             --build-arg BASE_DISTRO_IMAGE=$BASE_DISTRO_IMAGE \
             --build-arg BASE_DISTRO_VERSION=$BASE_DISTRO_VERSION \
             -f ./build_spec/cim_dockerfile \
             -t $MICROSERVICE_NAME:$MICROSERVICE_VERSION .

echo -e "\e[1;32;40m[CIM-BUILD] Setting Artifacts Environment \e[0m"
rm -rf $ARTIFACTS_PATH
mkdir -p $ARTIFACTS_PATH
mkdir -p $ARTIFACTS_PATH/images
mkdir -p $ARTIFACTS_PATH/config

echo -e "\e[1;32;40m[CIM-BUILD] Releasing Artifacts... @$ARTIFACTS_PATH \e[0m"
docker save $MICROSERVICE_NAME:$MICROSERVICE_VERSION | gzip > $ARTIFACTS_PATH/images/$MICROSERVICE_NAME-$MICROSERVICE_VERSION.tar.gz
cp -rf $CONFIG_RELEASE_LIST $ARTIFACTS_PATH/config/

echo -e "\e[1;32;40m[CIM-BUILD] Deleting Intermidiate Containers... \e[0m"
docker image prune -f --filter "label=IMAGE-TYPE=$BUILDER_LABEL"
docker rmi -f $MICROSERVICE_NAME:$MICROSERVICE_VERSION

if [[ -n "$DELETE_BUILDER_IMAGE" ]] && [[ "$DELETE_BUILDER_IMAGE" == "yes" ]]; then
echo -e "\e[1;32;40m[CIM-BUILD] Deleting Builder Image... \e[0m"
docker rmi -f $BUILDER_IMAGE:$BUILDER_VERSION
fi
