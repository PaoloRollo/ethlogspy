#!/bin/bash

cd /usr/local/ethlogspy
docker-compose build
nohup docker-compose up &