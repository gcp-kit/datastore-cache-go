package cache

// DeleteTiming - Controls the cache deletion timing.
//                  └── キャッシュの削除タイミングを制御する。
type DeleteTiming int

const (
	// DeleteTimingNone - Do not delete the cache.
	//                      └── キャッシュを削除しない。
	// It is not recommended because the cache may not be deleted even if operations such
	// as deletion and modification are performed, which may result in inconsistent data.
	//    └── 削除や変更などの操作を加えてもキャッシュが削除されず、データの整合性が取れなくなる恐れがあるため、非推奨。
	// Deprecated: Not recommended because the data integrity cannot be obtained.
	//    └── データの整合性が取れなくなるため非推奨
	DeleteTimingNone DeleteTiming = 0

	// DeleteTimingBeforeCommit - Execute cache deletion before issuing Commit.
	//                              └── Commitを発行する前にキャッシュの削除を実行する
	DeleteTimingBeforeCommit DeleteTiming = 1

	// DeleteTimingAfterCommit - Delete the cache after issuing Commit.
	//                             └── Commitを発行した後にキャッシュの削除を実行する
	DeleteTimingAfterCommit DeleteTiming = 2

	// DeleteTimingBeforeAndAfterCommit - Execute deletion before issuing Commit and twice after issuing Commit.
	//                                      └── Commitを発行する前と発行した後の2回に削除を実行する
	// Note that deletion is performed twice before commit and after commit,
	// and twice the normal query is sent to the cache.
	//    └── Commit前とCommit後の2回に削除が行われ、キャッシュに対して通常の倍のクエリが送信されるため注意
	DeleteTimingBeforeAndAfterCommit DeleteTiming = 3
)
