#!/bin/bash

GATE_ADDR="192.168.31.147:1025"
ETCD_ADDR="127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379"

# 创建任务，state应该是running
function create_and_check() {
  TASK_NAME="$1"
  if [[ -z "$TASK_NAME" ]];then
    TASK_NAME=`uuidgen`
  fi
  ./scheduler start -f ./example/http.yaml -g $GATE_ADDR -t "$TASK_NAME"

  sleep 1.5
  # -s 是global state task
  RESULT=`./scheduler etcd --get -s -t $TASK_NAME -e $ETCD_ADDR`

  # 输出执行的命令，方便debug
  echo "./scheduler etcd --get -s -t $TASK_NAME -e $ETCD_ADDR"

  check_have_result "$RESULT" "create_and_check"
}

function delete_and_check() {
  update_and_check_core "$1" "$2" "$3"
}

# 删除一个不存在的任务
function update_and_check_core() {

  TASK_NAME="$1"
  if [[ -z "$TASK_NAME" ]];then
    TASK_NAME=`uuidgen`
  fi

  FUNC_NAME="$2"
  if [[ -z "$FUNC_NAME" ]];then
    FUNC_NAME="delete_and_check"
  fi

  ACTION="$3"
  if [[ -z "$ACTION" ]];then
    ACTION="rm"
  fi
  echo "task_name($1), func_name($2) action($3)"
  # 生成task name
  
  ONLY="$4"
  ./scheduler $ACTION -f ./example/http.yaml -g $GATE_ADDR -t "$TASK_NAME"

  sleep 1
  # 查询全局队列是否有值
  RESULT=`./scheduler etcd --get -g -t $TASK_NAME -e $ETCD_ADDR`

  echo "./scheduler etcd --get -g -t $TASK_NAME -e $ETCD_ADDR"

  if [[ $ACTION = "rm" ]];then
    check_empty_result "$RESULT" $FUNC_NAME
  else 
    # 如果是先创建任务，再update或者stop，应该有值
    if [[ -z $ONLY ]];then
      RESULT=`./scheduler etcd --get -s -t $TASK_NAME -e $ETCD_ADDR`
      echo "./scheduler etcd --get -s -t $TASK_NAME -e $ETCD_ADDR"
      check_have_result "$RESULT" $FUNC_NAME
    else
      check_empty_result "$RESULT" $FUNC_NAME
    # 即是update或者删除，应该返回空结果
    fi
  fi
} 

# 先创建再删除
function create_and_delete_check() {
  TASK_NAME=`uuidgen`
  create_and_check $TASK_NAME
  delete_and_check $TASK_NAME "create_and_delete_check"
}

function create_and_stop_check() {
  TASK_NAME=`uuidgen`
  create_and_check $TASK_NAME
  update_and_check_core `uuidgen` "create_and_stop_check" "stop"
}

# 先创建再更新
function create_and_update_check() {
  TASK_NAME=`uuidgen`
  create_and_check $TASK_NAME
  update_and_check_core `uuidgen` "create_and_update_check" "update"
}

# 对一个不存在的任务更新，应该报错
function only_update_and_check() {
  update_and_check_core `uuidgen` "only_update_and_check" "update" "onlyupdate"
}

# stop一个不存在的任务，应该报错
function only_stop_and_check() {
  update_and_check_core `uuidgen` "only_stop_and_check" "stop" "onlystop"
}

# 检查create 之后的结果
function check_have_result() {
  result=`echo "$1"|grep running`
  if [[ ! -z $result ]];then
    echo -e "> $2 \033[32m check ok \033[0m, $1"
  else
    echo -e "> $2 \033[31m check fail \033[0m, $1"
    exit 1
  fi
}

# 检查删除之后的结果
function check_empty_result() {
  if [[ -z $1 ]];then
    echo -e "> $2 \033[32m check ok \033[0m, $1"
  else
    echo -e "> $2 \033[31m eheck fail \033[0m, $1"
    exit 1
  fi
}

# 先创建，再删除。
create_and_delete_check

# 先创建，再更新
create_and_stop_check

# 先创建，再更新
create_and_update_check

# 只创建，状态应该是running
create_and_check

# 删除一个不存在的任务，etcd里面数据应该是空的
delete_and_check `uuidgen`

# 更新一个不存在的任务，etcd里面数据应该是空的
only_update_and_check

# stop一个不存在的任务，etcd里面的数据应该是空的
only_stop_and_check

