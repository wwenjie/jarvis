#!/bin/bash

# 初始化配置服务器副本集
echo "Initializing config server replica set..."
sudo docker exec mongod_configsvr mongosh --port 27018 --eval '
rs.initiate({
  _id: "config",
  members: [
    {_id: 0, host: "10.1.20.17:27018"}
  ]
})
'

# 每个分片服务器初始化副本集
echo "Initializing shard replica set..."
for i in 1 2 3; do
    port=$((27018 + $i))
    sudo docker exec mongod_shard$i mongosh --port $port --eval "
rs.initiate({
  _id: \"shard$i\",
  members: [
    {_id: 0, host: \"10.1.20.17:$port\"}
  ]
})
"
done

# 添加分片到集群
echo "Adding shards to cluster..."
sudo docker exec mongos mongosh --port 27017 --eval '
sh.addShard("shard1/10.1.20.17:27019")
sh.addShard("shard2/10.1.20.17:27020")
sh.addShard("shard3/10.1.20.17:27021")
'

echo "MongoDB cluster initialization completed!" 