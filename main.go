package main

import (
	"fmt"
	"time"

	"github.com/velar-agent/alarm-clock/busywait"
)

const timestampLayout = "2006-01-02 15:04:05.000000000 -07:00"

func main() {
	// 표시 시각과 경과 시간의 기준을 벽시계로 통일합니다.
	startedAt := time.Now().Round(0)
	alarmAt := nextMinute(startedAt)
	clock := busywait.NewAlarmClock()

	var firedAt time.Time
	done := clock.SetAlarm(alarmAt, func() {
		// 출력 지연이 측정에 섞이지 않도록 콜백 진입 직후 시각을 기록합니다.
		firedAt = time.Now()
	})

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	fmt.Printf("Started at:   %s\n", startedAt.Format(timestampLayout))
	fmt.Printf("Alarm target: %s\n", alarmAt.Format(timestampLayout))
	fmt.Printf("Counter: %2ds | Now: %s\n", 0, startedAt.Format(timestampLayout))

	for {
		select {
		case <-done:
			// 완료 채널을 받은 뒤에는 콜백이 기록한 시각을 안전하게 읽을 수 있습니다.
			lateness := firedAt.Sub(alarmAt)
			fmt.Printf("Alarm fired:  %s\n", firedAt.Format(timestampLayout))
			fmt.Printf("Elapsed:      %s\n", firedAt.Sub(startedAt))
			fmt.Printf("Lateness:     %s (%d ns)\n", lateness, lateness.Nanoseconds())
			return
		case <-ticker.C:
			now := time.Now()
			elapsedSeconds := int64(now.Sub(startedAt) / time.Second)
			fmt.Printf("Counter: %2ds | Now: %s\n", elapsedSeconds, now.Format(timestampLayout))
		}
	}
}

func nextMinute(startedAt time.Time) time.Time {
	elapsedInMinute := time.Duration(startedAt.Second())*time.Second +
		time.Duration(startedAt.Nanosecond())
	// 분 정각은 벽시계 기준이므로 단조 시계 정보를 제거합니다.
	return startedAt.Add(time.Minute - elapsedInMinute).Round(0)
}
