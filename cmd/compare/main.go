package main

import (
	"fmt"
	"time"

	"github.com/velar-agent/alarmclock"
)

const timestampLayout = "2006-01-02 15:04:05.000000000 -07:00"

func main() {
	// 목표와 측정값을 모두 벽시계 기준으로 맞춥니다.
	startedAt := time.Now().Round(0)
	alarmAt := nextMinute(startedAt)
	reportAt := alarmAt.Add(time.Second)

	var busyFiredAt, afterFiredAt time.Time

	// 바쁜대기를 시작하기 전에 일반 타이머를 등록합니다.
	after := time.After(time.Until(alarmAt))
	afterDone := make(chan struct{})
	go func() {
		<-after
		// 채널이 전달한 시각 대신 실제 수신 직후 시각을 기록합니다.
		afterFiredAt = time.Now().Round(0)
		close(afterDone)
	}()

	busyDone := alarmclock.New().SetAlarm(alarmAt, func() {
		// 콜백에 진입한 직후 시각을 기록합니다.
		busyFiredAt = time.Now().Round(0)
	})

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	reportTimer := time.NewTimer(time.Until(reportAt))
	defer reportTimer.Stop()

	fmt.Printf("Started at:   %s\n", startedAt.Format(timestampLayout))
	fmt.Printf("Alarm target: %s\n", alarmAt.Format(timestampLayout))
	fmt.Printf("Report target: %s\n\n", reportAt.Format(timestampLayout))
	fmt.Printf("Counter: %2ds\n", 0)

	for {
		select {
		case <-reportTimer.C:
			// 정분 1초 뒤에 완료 채널로 측정값 기록을 확인하고 출력합니다.
			<-busyDone
			<-afterDone

			fmt.Printf("\nCompared at: %s\n", time.Now().Format(timestampLayout))
			fmt.Printf("%-10s | %-36s | %s\n", "Method", "Fired at", "Lateness")
			printResult("busywait", busyFiredAt, alarmAt)
			printResult("time.After", afterFiredAt, alarmAt)

			difference := afterFiredAt.Sub(alarmAt) - busyFiredAt.Sub(alarmAt)
			fmt.Printf("Difference (time.After - busywait): %s (%d ns)\n",
				difference, difference.Nanoseconds())
			// 환경 조회가 알람 측정에 영향을 주지 않도록 결과 출력 뒤에 수행합니다.
			printEnvironment()
			return
		case <-ticker.C:
			now := time.Now()
			elapsedSeconds := int64(now.Sub(startedAt) / time.Second)
			fmt.Printf("Counter: %2ds\n", elapsedSeconds)
		}
	}
}

func printResult(name string, firedAt, alarmAt time.Time) {
	lateness := firedAt.Sub(alarmAt)
	fmt.Printf("%-10s | %s | %s (%d ns)\n",
		name, firedAt.Format(timestampLayout), lateness, lateness.Nanoseconds())
}

func nextMinute(startedAt time.Time) time.Time {
	elapsedInMinute := time.Duration(startedAt.Second())*time.Second +
		time.Duration(startedAt.Nanosecond())
	return startedAt.Add(time.Minute - elapsedInMinute).Round(0)
}
