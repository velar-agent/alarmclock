package alarmclock

import "time"

type AlarmClock struct {
	// 목표 시각 전에 바쁜대기를 수행할 시간입니다.
	spinWindow time.Duration
}

// New는 바쁜대기 구간이 1분인 알람 시계를 생성합니다.
// 알람1분전에 CPU가 깨어남
func New() *AlarmClock {
	return &AlarmClock{
		spinWindow: time.Minute,
	}
}

// SetAlarm은 목표 시각까지 기다린 뒤 action을 실행하고 완료 채널을 닫습니다.
func (clock *AlarmClock) SetAlarm(at time.Time, action func()) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		// 바쁜대기 구간이 시작될 때까지 잠들어 CPU 사용을 줄입니다.
		if remaining := time.Until(at); remaining > clock.spinWindow {
			time.Sleep(remaining - clock.spinWindow)
		}

		// 바쁜대기 구간에서는 목표 시각에 도달할 때까지 시간을 계속 확인합니다.
		for time.Until(at) > 0 {
		}

		// 기다리던 고루틴에서 작업을 바로 실행합니다.
		action()

		// 작업이 끝난 뒤 완료를 알립니다.
		close(done)
	}()

	return done
}
