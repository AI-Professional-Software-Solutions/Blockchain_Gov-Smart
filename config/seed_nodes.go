package config

type SeedNode struct {
	Url string `json:"url" msgpack:"url"`
}

var (
	MAIN_NET_SEED_NODES = []*SeedNode{}

	TEST_NET_SEED_NODES = []*SeedNode{
		{
			"wss://blockchain.gov-smart.com:2053/ws",
		},
		{
			"wss://blockchain.gov-smart.com:2083/ws",
		},
		{
			"wss://blockchain.gov-smart.com:2087/ws",
		},
		{
			"wss://blockchain.gov-smart.com:2096/ws",
		},
	}

	DEV_NET_SEED_NODES = []*SeedNode{
		{
			"wss://blockchain.gov-smart.com:2053/ws",
		},
		{
			"wss://blockchain.gov-smart.com:2083/ws",
		},
		{
			"wss://blockchain.gov-smart.com:2087/ws",
		},
		{
			"wss://blockchain.gov-smart.com:2096/ws",
		},
	}
)
