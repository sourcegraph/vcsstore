package cluster

import "net/url"

func key(urlStr string) CloneURL {
	u, err := url.Parse(urlStr)
	if err != nil {
		panic("key: " + err.Error())
	}
	return CloneURL{"git", u}
}
