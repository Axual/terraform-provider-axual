#!/bin/sh
# Replace all references of stream in state files
sed -i='' "s/axual_stream_config/axual_topic_config/" terraform.tfstate
sed -i='' "s/axual_stream/axual_topic/" terraform.tfstate
sed -i='' "s/\"stream\":/\"topic\":/" terraform.tfstate
# Replace all references of stream in configuration files
sed -i='' "s/axual_stream_config/axual_topic_config/" main.tf
sed -i='' "s/axual_stream/axual_topic/" main.tf
sed -i='' "s/stream =/topic =/" main.tf
