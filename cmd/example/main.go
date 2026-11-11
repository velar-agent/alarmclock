package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/velar-agent/alarmclock"
)

func main() {
	const alarmCount, workerCount = 3, 8

	clock := alarmclock.New()
	base := time.Now()
	var firedAt [alarmCount][workerCount]time.Time
	var done sync.WaitGroup

	// Register alarms for 10, 20, and 30 seconds from the same base time.
	// 같은 기준 시각으로부터 10초, 20초, 30초 뒤 알람을 등록합니다.
	for alarmID := 1; alarmID <= alarmCount; alarmID++ {
		at := base.Add(time.Duration(alarmID) * 10 * time.Second)

		// All eight workers register the exact same target for this alarm.
		// 이 알람의 작업자 8개는 모두 정확히 같은 목표 시각을 등록합니다.
		for workerID := 1; workerID <= workerCount; workerID++ {
			ch := clock.SetAlarm(at)
			done.Add(1)

			go func(alarmID, workerID int) {
				defer done.Done()
				<-ch
				receivedAt := time.Now()
				firedAt[alarmID-1][workerID-1] = receivedAt
			}(alarmID, workerID)
		}
	}

	done.Wait()

	// Format and print only after every worker has recorded its receive time.
	// 모든 작업자가 수신 시각을 기록한 뒤에만 문자열 변환과 출력을 수행합니다.
	for alarmIndex, workers := range firedAt {
		for workerIndex, receivedAt := range workers {
			fmt.Printf("Alarm %d / worker %d: %s\n",
				alarmIndex+1, workerIndex+1, receivedAt.Format("15:04:05.000000"))
		}
	}
}
