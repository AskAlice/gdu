//go:build windows || plan9

package indexer

func deviceID(path string) uint64 { return 0 }
