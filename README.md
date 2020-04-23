## datastore cache
datastoreのgRPCをフックし、datastoreのキャッシュを実現する。  
キャッシュ機構は、middlewareとして途中に噛ませることで、容易に入れ替えが可能なようにする。

キャッシュは、gRPCの通信をmiddlewareで操作している。キャッシュで使っているgRPCのメソッドは、 `/google.datastore.v1.Datastore/Lookup` と `/google.datastore.v1.Datastore/Commit` の2つ。 デフォルト動作では、すべての要素がキャッシュされ、すべての要素が更新時にキャッシュ削除が行われる。これらの動作はオプションにより変更可能。  
また、要素を削除する際の挙動は、 `CacheDeleteTiming` で変更可能。デフォルトの動作は、`DeleteTimingBeforeAndAfterCommit` で、要素の操作前と操作後の2回に行われる。この場合、通常の倍のクエリが発行される。 `CacheDeleteTiming` の詳細は、godocを参照。
キャッシュ挙動の変更するには、初期化時に `CachingModeFunc` を設定しし、任意の値を返すことで動作を変えることができる。 `CachingModeFunc` はgRPCのMiddlewareと同等の引数が渡される。

現状提供しているキャッシュは、Redisによるキャッシュ。追加する場合は、ライブラリ内にあるcacheインターフェイスを満たすものを作成する事が必要。

### cache interface
cacheインターフェイスは、 `datastore/cache.go` にあるインターフェイス。  
このインターフェイスを満たすものを、ミドルウェアとして設定することで、様々な機構によるキャッシュを実現する。
詳細な内容・使い方は `cache.go` 内のgodocを参照。

### redis cache
redis cacheは、 `cache` インターフェイスを満たす構造体。   
`datastore/cache` と合わせて使うことでRedisによるキャッシュを実現することができる。  

redisでは、keyにdatastoreのentityのkey.pathをシリアライズしたものを使う。  

### コード記述例
```go
// redis(redigo)のpoolを作成
pool := &redigo.Pool{
	MaxIdle:     3,
	MaxActive:   0,
	IdleTimeout: 240 * time.Second,
	Dial:        func() (redigo.Conn, error) { 
		return redigo.Dial("tcp", addr) 
	},
}

// cache middlewareを作成
middleware := cache.NewMiddleware(redis.NewRedis(pool))

// 特定の要素ではキャッシュしない・キャッシュの削除のタイミングを制御などの細かい操作を行う。
// cache.CachingModeTypeによってキャッシュの動作を変更する。 詳細はgodocを参照。
// middleware.CachingModeFuncをnilにした場合はデフォルトの動作。
middleware.CachingModeFunc = func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) cache.CachingModeType {
	// some code
}


// datastoreのClientを初期化時に、cache middlewareを渡すことでgRPCの通信をフックさせる
client, _ := datastore.NewClient(
	context.Background(),
	"",
	// ここでgRPCの通信をMiddlewareへフックさせるようにする
	option.WithGRPCDialOption(
		grpc.WithUnaryInterceptor(middleware.UnaryClientInterceptor),
	),
)


ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

key := datastore.IncompleteKey("test", nil)
data := &TestData{Name: "foo"}

// 任意の値を挿入。
key, _ := client.Put(ctx, key, data)

var ret TestData
// 指定した値を取り出す
client.Get(ctx, key, &ret)
```
※ サンプルコードのため、エラーハンドリングは省略
