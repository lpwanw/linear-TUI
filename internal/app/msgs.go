package app

import "time"

// Internal messages emitted by the app itself (not the sync service).

type MotionTimeoutMsg struct{ At time.Time }

type BannerClearMsg struct{}

type QuitMsg struct{}
