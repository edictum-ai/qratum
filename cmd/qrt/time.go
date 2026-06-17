package main

import "time"

func currentTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
