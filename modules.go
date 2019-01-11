package main

import (
	_ "github.com/henryse/registrator/consul"
	_ "github.com/henryse/registrator/consulkv"
	_ "github.com/henryse/registrator/etcd"
	_ "github.com/henryse/registrator/skydns2"
	_ "github.com/henryse/registrator/zookeeper"
)
