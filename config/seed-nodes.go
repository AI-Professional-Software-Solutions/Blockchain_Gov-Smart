package config

import "net/url"

var (
	MAIN_NET_SEED_NODES = []url.URL{
		{Scheme: "ws", Host: "127.0.0.1:5231", Path: "/ws"},
	}

	TEST_NET_SEED_NODES = []url.URL{
		{Scheme: "ws", Host: "helloworldx.ddns.net:6000", Path: "/ws"},
		{Scheme: "ws", Host: "helloworldx.ddns.net:6001", Path: "/ws"},
		{Scheme: "ws", Host: "helloworldx.ddns.net:6002", Path: "/ws"},
		{Scheme: "ws", Host: "helloworldx.ddns.net:6003", Path: "/ws"},
	}

	DEV_NET_SEED_NODES = []url.URL{
		{Scheme: "ws", Host: "127.0.0.1:5231", Path: "/ws"},
	}
)
