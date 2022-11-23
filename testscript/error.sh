# 统计出错日志，以及出错次数
echo "#######error#######"
grep error cluster.log|grep '{.*}' -o|jq '.message'|sort|uniq -c

echo "#######warn#######"
grep warn cluster.log|grep '{.*}' -o|jq '.message'|sort|uniq -c
