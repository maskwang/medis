package main

import (
	"log"
	"medis/adapter"
	"medis/admin"
	"medis/datasource"
	"medis/mysql"
	"medis/proxy"
	"medis/shard"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	// 性能采集接口
	go func() {
		http.ListenAndServe(":13800", nil)
	}()
	// 管理后台接口
	admin.NewAdminServer(":13801")
	// 创建mysql元组
	master0, err := mysql.NewMysqlClient("root", "root", "localhost", 8889, "test", "utf8")
	if err != nil {
		log.Fatal(err)
	}
	master1, err := mysql.NewMysqlClient("root", "root", "localhost", 8889, "test1", "utf8")
	if err != nil {
		log.Fatal(err)
	}
	// 创建主从关系
	group0 := datasource.NewGroup("group0")
	// 0 1 0 1 代表的是 读权重是0，写权重是1，不可读，写优先级是1
	group0.AddClient(datasource.NewClientWeightWrapper("group0_master_0", master0, 0, 1, 0, 1))
	group0.AddClient(datasource.NewClientWeightWrapper("group0_slave_0", master0, 1, 0, 1, 0))
	group0.Init()
	group1 := datasource.NewGroup("group1")
	group1.AddClient(datasource.NewClientWeightWrapper("group1_master_0", master1, 0, 1, 0, 1))
	group1.AddClient(datasource.NewClientWeightWrapper("group1_slave_0", master1, 1, 0, 1, 0))
	group1.Init()
	// 创建sharding关系
	selector := shard.NewSelector("test_selector")
	selector.AddGroup(group0)
	selector.AddGroup(group1)
	admin.RegisterSelector(selector)
	// 创建REDIS到数据库的适配器
	dbAdapter, _ := adapter.NewDBAdapter(selector)
	// 创建REDIS协议的Handler
	handler, _ := proxy.NewMedisHandler(dbAdapter)
	// 启动REDIS
	proxy.ListenAndServeRedis(handler)
}
