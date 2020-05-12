# What is this?
Hook Datastore gRPC to implement Datastore cache.  
The cache contents makes it possible to easily replace it by biting it in the middle as middleware.
 
The cache gRPC communication is operating with a middleware.  
There are two gRPC methods used in the cache: `/google.datastore.v1.Datastore/Lookup` and`/google.datastore.v1.Datastore/Commit`.  
The default behavior is that all elements will be cached and all elements will be cache delete upon updated.  
These operations can be changed by options.  

 Also, the behavior when deleting an element can be changed with `CacheDeleteTiming`.  
 The default behavior is `DeleteTimingBeforeAndAfterCommit`, which is done twice before and after the operation of the element.  
 In this case, twice as many queries are issued as usual.  
 For details of `CacheDeleteTiming`, refer to godoc.  
 
 To change the cache behavior, you can change the behavior by setting `CachingModeFunc` at initialization and returning an arbitrary value.  
 The argument equivalent to Middleware of gRPC is passed to `CachingModeFunc`.  
 
 The current provided is cache by Redis.  
 When adding, it is necessary to create one that satisfies the Cache interface in the library.  

 ## Installation
 ```commandline
 go get -u github.com/gcp-kit/datastore-cache-go
 ```

 ## Cache interface
 Cache interface is an interface in `cache/cache.go`.  
 By setting what satisfies this interface as middleware, the cache by various mechanisms is realized.  
 See godoc in `cache.go` for detailed contents and usage.  
 
 ## Redis cache
 Redis cache is a structure that satisfies the `cache` interface.  
 Cache with Redis can be realized by using together with `datastore-cache-go/cache`.  
 
In Redis, use the serialized `key.path` of the Datastore Entity as the key.  
 
## Usage
```go
import (
	"context"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/gcp-kit/datastore-cache-go/cache"
	"github.com/gcp-kit/datastore-cache-go/cache/redis"
	redigo "github.com/gomodule/redigo/redis"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

// create a pool of redis(redigo)
pool := &redigo.Pool{
	MaxIdle:     3,
	MaxActive:   0,
	IdleTimeout: 240 * time.Second,
	Dial:        func() (redigo.Conn, error) { 
		return redigo.Dial("tcp", addr) 
	},
}

// create cache middleware
middleware := cache.NewMiddleware(redis.NewRedis(pool))

// Perform detailed operations such as not caching at a specific element and controlling the cache deletion timing.
// Change cache behavior by cache.CachingModeType. See godoc for details.
// Default behavior when middleware.Caching Mode Func is set to nil.
middleware.CachingModeFunc = func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) cache.CachingModeType {
	// some code
}


// Hook gRPC communication by passing cache middleware when initializing Datastore Client.
client, _ := datastore.NewClient(
	context.Background(),
	"",
	// Here, make gRPC communication hooked to Middleware.
	option.WithGRPCDialOption(
		grpc.WithUnaryInterceptor(middleware.UnaryClientInterceptor),
	),
)


ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

key := datastore.IncompleteKey("test", nil)
data := &TestData{Name: "foo"}

// Insert any value.
key, _ := client.Put(ctx, key, data)

var ret TestData
// Get specified value
client.Get(ctx, key, &ret)
```
Since it is a sample code, error handling is omitted.

## Notes
When testing locally, run the following to launch the emulator and Redis.
```commandline
gcloud beta emulators datastore start --project=projectName --host-port 0.0.0.0:8000 --no-store-on-disk
```
```commandline
redis-server /path/to/redis.conf --loglevel verbose
```