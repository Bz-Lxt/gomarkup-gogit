package git

import "time"

var beijing = time.FixedZone("GMT+8", 8*3600)

func formatBeijing(unix int64) string {
	if unix <= 0 {
		return ""
	}
	return time.Unix(unix, 0).In(beijing).Format("2006-01-02 15:04:05")
}

func NowBeijing() time.Time {
	return time.Now().In(beijing)
}

func BeijingLocation() *time.Location {
	return beijing
}
