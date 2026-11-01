package busywait

import "time"

// 알람1분전에 CPU가 깨어남
const spinWindow = time.Minute

type AlarmClock struct{}

func NewAlarmClock() *AlarmClock {
	return &AlarmClock{}
}

// SetAlarm은 목표 시각까지 기다린 뒤 action을 실행하고 완료 채널을 닫습니다.
func (*AlarmClock) SetAlarm(at time.Time, action func()) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		// 목표 시각 1분 전까지는 잠들어 CPU 사용을 줄입니다.
		if remaining := time.Until(at); remaining > spinWindow {
			time.Sleep(remaining - spinWindow)
		}

		// 마지막 1분 동안은 목표 시각에 도달할 때까지 시간을 계속 확인합니다.
		for time.Until(at) > 0 {
		}

		// 기다리던 고루틴에서 작업을 바로 실행합니다.
		action()

		// 작업이 끝난 뒤 완료를 알립니다.
		close(done)
	}()

	return done
}
