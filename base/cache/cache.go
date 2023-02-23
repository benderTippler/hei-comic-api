package cache

import "github.com/gogf/gf/v2/os/gcache"

var (
	CacheManager *gcache.Cache
)

func init() {
	CacheManager = gcache.New() // 定义全局缓存对象，使用内存缓存
}
