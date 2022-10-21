#!/bin/bash

function create() {
  # TODO -g的地址自动获取一个
  ./scheduler start -f ./example/http.yaml -g 192.168.31.147:1025 -d
}

function delete() {
  ./scheduler rm -f ./example/http.yaml -g 192.168.31.147:1025 -d
}

function create_and_delete() {
  create
  delete
}

echo "($1)" "($2)" "($3)"
if [[ $1 = "create" ]];then
  create
  exit 0
fi
create_and_delete
