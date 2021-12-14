package log

import "time"

var (
	// 时区设置，参考 https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
	Local  = time.Local
	UTC  = time.UTC
	Beijing,_  = time.LoadLocation("Asia/Shanghai")
	LosAngeles,_  = time.LoadLocation("America/Los_Angeles")

)