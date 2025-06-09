# Docker Compose 配置

本目录包含各种服务的Docker Compose配置文件。以下是每个文件的简要介绍和使用说明。

## etcd-docker-compose.yml

etcd集群。

## service-docker-compose.yml

jarvis后台服务。包括api_gateway和rag_svr

## milvus-docker-compose.yml

Milvus向量数据库单例。

## redis-docker-compose.yml

Redis集群。支持宿主机内多redis节点和跨宿主机的集群。本容器配置是用的host网络模式，而redis-docker-compose.yml.bak是用的桥接模式。

## mongodb-docker-compose.yml

MongoDB数据库集群。1个configsvr，1个mongos，3个mongod分片

## mysql-docker-compose.yml

mysql单例。
