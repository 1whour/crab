#!/bin/bash

GATE_ADDR="192.168.31.147:1025"
ETCD_ADDR="127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379"

function create() {
  TASK_NAME="$1"
  ./scheduler start -f ./example/http.yaml -g $GATE_ADDR -t "$TASK_NAME"
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

  check_result "$result" $FUNC_NAME
} 

function create_and_delete_check() {
  TASK_NAME=`uuidgen`
  create $TASK_NAME
  delete_and_check $TASK_NAME "create_and_delete"
}

function check_result() {
  if [[ -z $1 ]];then
    echo "> $2 check ok, $1"
  else
    echo "> $2 eheck fail"
    exit 1
  fi

}

delete_and_check
create_and_delete_check
