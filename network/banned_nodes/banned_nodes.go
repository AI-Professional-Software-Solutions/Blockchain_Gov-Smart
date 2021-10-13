package banned_nodes

import (
	"net/url"
	"sync"
	"time"
)

type BannedNode struct {
	URL        *url.URL
	Timestamp  time.Time
	Expiration time.Time
	Message    string
}

type BannedNodes struct {
	bannedMap *sync.Map //map[string]
}

func (self *BannedNodes) IsBanned(urlStr string) bool {
	if _, found := self.bannedMap.Load(urlStr); found {
		return true
	}
	return false
}

func (self *BannedNodes) Ban(url *url.URL, urlStr, message string, duration time.Duration) {
	if urlStr == "" {
		urlStr = url.String()
	}
	time := time.Now()
	self.bannedMap.Store(urlStr, &BannedNode{
		URL:        url,
		Message:    message,
		Timestamp:  time,
		Expiration: time.Add(duration),
	})
}

func CreateBannedNodes() *BannedNodes {
	return &BannedNodes{
		bannedMap: &sync.Map{},
	}
}