package cache

// DeleteTiming - キャッシュの削除タイミングを制御する。
type DeleteTiming int

const (
	// DeleteTimingNone - キャッシュを削除しない。
	// 削除や変更などの操作を加えてもキャッシュが削除されないため、データの整合性が取れなくなる恐れがあるため、非推奨。
	// Deprecated: データの整合性が取れなくなるため非推奨。
	DeleteTimingNone DeleteTiming = 0

	// DeleteTimingBeforeCommit - Commitを発行する前にキャッシュの削除を実行する。
	DeleteTimingBeforeCommit DeleteTiming = 1

	// DeleteTimingAfterCommit - Commitを発行した後にキャッシュの削除を実行する。
	DeleteTimingAfterCommit DeleteTiming = 2

	// DeleteTimingBeforeAndAfterCommit - Commitを発行する前と発行した後の2回に削除を実行する。
	// Commit前とCommit後の2回に削除が行われ、キャッシュに対して通常の倍のクエリが送信されるため、注意。
	DeleteTimingBeforeAndAfterCommit DeleteTiming = 3
)
