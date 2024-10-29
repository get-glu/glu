package containers

import "iter"

// FilterByKey filters the provided sequence using the provided key function
// The result is a sequence containing only the entries where the key in the
// supplied sequence returns true when passed to keyFn
func FilterByKey[K comparable, V any](seq iter.Seq2[K, V], keyFn func(K) bool) iter.Seq2[K, V] {
	return iter.Seq2[K, V](func(yield func(K, V) bool) {
		for k, v := range seq {
			if !keyFn(k) {
				continue
			}

			if !yield(k, v) {
				return
			}
		}
	})
}
