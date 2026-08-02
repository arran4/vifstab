package rules_test

import (
	"github.com/arran4/vifstab/internal/rules"
	"testing"
	"testing/fstest"
)

func BenchmarkCheckGroupExists(b *testing.B) {
	mockFS := fstest.MapFS{
		"etc/group": &fstest.MapFile{
			Data: []byte("root:x:0:\nbin:x:1:\ndaemon:x:2:\nsys:x:3:\nfoo:x:1000:\n"),
		},
	}
	cache := rules.NewCache()
	// Note: We're calling string cast so that the cache actually stores the boolean and we can fetch it.
	cacheFS := rules.NewCacheFS(cache, mockFS)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cacheFS.CheckGroupExists(1000)
		cacheFS.CheckGroupExists(9999)
	}
}

func BenchmarkCheckUserExists(b *testing.B) {
	mockFS := fstest.MapFS{
		"etc/passwd": &fstest.MapFile{
			Data: []byte("root:x:0:0:root:/root:/bin/bash\nbin:x:1:1:bin:/bin:/sbin/nologin\ndaemon:x:2:2:daemon:/sbin:/sbin/nologin\nsys:x:3:3:sys:/dev:/sbin/nologin\nfoo:x:1000:1000:foo:/home/foo:/bin/bash\n"),
		},
	}
	cache := rules.NewCache()
	cacheFS := rules.NewCacheFS(cache, mockFS)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cacheFS.CheckUserExists(1000)
		cacheFS.CheckUserExists(9999)
	}
}
