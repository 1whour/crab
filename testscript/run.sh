#!/bin/bash

GATE_ADDR="192.168.31.147:1025"
ETCD_ADDR="127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379"

function create_and_check() {
  TASK_NAME="$1"
  if [[ -z "$TASK_NAME" ]];then
    TASK_NAME=`uuidgen`
  fi
  ./scheduler start -f ./example/http.yaml -g $GATE_ADDR -t "$TASK_NAME"

  sleep 1
  # -s 是global state task
  result=`./scheduler etcd --get -s -t $TASK_NAME -e $ETCD_ADDR`

  # 输出执行的命令，方便debug
  echo "./scheduler etcd --get -s -t $TASK_NAME -e $ETCD_ADDR --debug"

  check_create_result "$result" "create_and_check"
}

# 删除一个不存在的任务，不应该在etcd里面写入数据
function delete_and_check() {

  TASK_NAME="$1"
  if [[ -z "$TASK_NAME" ]];then
    TASK_NAME=`uuidgen`
  fi

  FUNC_NAME="$2"
  if [[ -z "$FUNC_NAME" ]];then
    FUNC_NAME="delete_and_check"
  fi

  # 生成task name
  # 删除
  ./scheduler rm -f ./example/http.yaml -g $GATE_ADDR -t "$TASK_NAME"

  # 查询全局队列是否有值
  result=`./scheduler etcd --get -g -t $TASK_NAME -e $ETCD_ADDR`

  echo "./scheduler etcd --get -g -t $TASK_NAME -e $ETCD_ADDR"

  check_delete_result "$result" $FUNC_NAME
} 

#
function create_and_delete_check() {
  TASK_NAME=`uuidgen`
  create $TASK_NAME
  delete_and_check $TASK_NAME "create_and_delete"
}

function check_create_result() {
  result=`echo "$1"|grep running`
  if [[ ! -z $result ]];then
    echo -e "> $2 \033[32m check ok \033[0m, $1"
  else
    echo -e "> $2 \033[31m eheck fail \033[0m, $1"
    exit 1
  fi
}

function check_delete_result() {
  if [[ -z $1 ]];then
    echo -e "> $2 \033[32m check ok \033[0m, $1"
  else
    echo -e "> $2 \033[31m eheck fail \033[0m"
    exit 1
  fi
}

create_and_check
delete_and_check
#create_and_delete_check
