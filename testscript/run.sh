#!/bin/bash

# 配置地址
GATE_ADDR="192.168.31.147:1025"
ETCD_ADDR="127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379"
MOCK_ADDR="http://127.0.0.1:8181"

source "./testscript/assert.sh"
function update_gate_addr() {
  GATE_ADDR=`etcdctl get --prefix /scheduler/v1/node/gate/ --print-value-only|head -1|tr -d '\n'`
}
# 创建任务，state应该是running
function create_and_check() {
  update_gate_addr

  TASK_NAME="$1"
  if [[ -z "$TASK_NAME" ]];then
    TASK_NAME=`uuidgen`
  fi
  FILE_NAME="$2"
  if [[ -z "$FILE_NAME" ]];then
    FILE_NAME="./example/http.yaml"
  fi

  CMD="./scheduler start -f $FILE_NAME -g $GATE_ADDR -t $TASK_NAME"
  echo $CMD
  `$CMD`
  assert_eq $? 0 "更新失败"

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

  update_gate_addr

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
  update_and_check_core $TASK_NAME "create_and_stop_check" "stop"
}

# 先创建再更新
function create_and_update_stop_check() {
  TASK_NAME=`uuidgen`
  create_and_check $TASK_NAME
  update_and_check_core $TASK_NAME "create_and_update_stop_check" "update"
  update_and_check_core $TASK_NAME "create_and_update_stop_check" "stop"
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

# 检查任务是运行的次数是否满足预期
function create_and_check_running_count() {
  TASK_NAME=`uuidgen`
  create_and_check $TASK_NAME
  sleep 3
  # 获取运行的次数
  CMD="curl -s -X GET -H scheduler-http-executer:$TASK_NAME $MOCK_ADDR/task"
  # 打印命令，方便debug用的
  echo $CMD
  #运行命令
  NUM=`$CMD`
  assert_ge $NUM 2 "任务执行次数太少 $NUM"
  assert_le $NUM 4 "任务执行次数太多 $NUM"
  update_and_check_core $TASK_NAME "create_and_stop_check" "stop"
}

# 检查故障转移功能
# 集群一开始启动两个gate, 关闭有任务在运行的gate
function failover_gate() {
  # 默认会起两个gate节点，先关闭gate1，那任务肯定会跑在gate2上面
  goreman run stop scheduler.gate1
  TASK_NAME=`uuidgen`

  # 运行任务
  create_and_check $TASK_NAME

  # 恢复gate1节点
  goreman run start scheduler.gate1
  # 关闭gate2节点
  goreman run stop scheduler.gate2

  # 检查运行次数是否满足预期
  sleep 3
  # 获取运行的次数
  CMD="curl -s -X GET -H scheduler-http-executer:$TASK_NAME $MOCK_ADDR/task"
  # 打印命令，方便debug用的
  echo $CMD
  #运行命令
  NUM=`$CMD`
  assert_ge $NUM 2 "任务执行次数太少 $NUM"
  assert_le $NUM 4 "任务执行次数太多 $NUM"
  update_and_check_core $TASK_NAME "failover_gate_stop" "stop"

  # 恢复gate2节点
  goreman run start scheduler.gate2
}

# 检查故障转移功能
# 集群一开始启动两个runtime, 关闭一个有任务的runtime
function failover_runtime() {
  echo ""
}

# 集群被重启了
# 需要做任务的自动恢复
function restart_cluster_resume_task() {
  echo ""
}

# 测试shell任务
create_and_check_shell_count() {
  TASK_NAME=`uuidgen`
  cp ./example/shell.yaml ./example/tmp_shell.yaml
  sed -i '' "s/TEMPLATE_VALUE/$TASK_NAME/" ./example/tmp_shell.yaml 
  create_and_check $TASK_NAME "./example/tmp_shell.yaml"
  sleep 3
  # 获取运行的次数
  CMD="curl -s -X GET -H scheduler-http-executer:$TASK_NAME $MOCK_ADDR/task"
  # 打印命令，方便debug用的
  echo $CMD
  #运行命令
  NUM=`$CMD`
  assert_ge $NUM 2 "任务执行次数太少 $NUM"
  assert_le $NUM 4 "任务执行次数太多 $NUM"
  update_and_check_core $TASK_NAME "create_and_stop_shell_check" "stop"

}

# 测试gate被重启是否能恢复任务
failover_gate

## 测试shell任务
#create_and_check_shell_count

## 测试任务是否能正确执行
#create_and_check_running_count
#
## 先创建，再更新
#create_and_stop_check
# 
## 先创建，再删除。
#create_and_delete_check
# 
## # 先创建，再更新
#create_and_update_stop_check
#
## 删除一个不存在的任务，etcd里面数据应该是空的
#delete_and_check `uuidgen`
#
## 更新一个不存在的任务，etcd里面数据应该是空的
#only_update_and_check
#
## stop一个不存在的任务，etcd里面的数据应该是空的
#only_stop_and_check
#
