package main

import "time"

var nowUTC = func() time.Time {
	return time.Now().UTC()
}

func currentTimestamp() string {
	return nowUTC().Format(time.RFC3339Nano)
}
