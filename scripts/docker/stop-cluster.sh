#!/bin/bash

# Docker Redis集群停止脚本
echo "停止Redis集群..."

# 停止Redis集群服务
docker-compose stop redis-master-1 redis-master-2 redis-master-3 redis-slave-1 redis-slave-2 redis-slave-3 redis-cluster-init

# 删除容器
docker-compose rm -f redis-master-1 redis-master-2 redis-master-3 redis-slave-1 redis-slave-2 redis-slave-3 redis-cluster-init

echo "Redis集群已停止！"