#!/bin/bash

http put localhost:8500/v1/catalog/deregister ServiceID=$1 Datacenter=dc1  Node=foobar
