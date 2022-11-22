# 统计出错日志，以及出错次数
grep error cluster.log|grep '{.*}' -o|jq '.message'|sort|uniq -c
