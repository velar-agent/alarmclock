package alarmclock

import (
	"sync"
	"sync/atomic"
	"time"
)

const alarmChannelCount = 4

type AlarmClock struct {
	spinWindow time.Duration
	mu         sync.Mutex
	alarms     map[time.Time]*alarmGroup
}

type alarmGroup struct {
	channels   [alarmChannelCount]chan struct{}
	next       int
	spinWindow atomic.Int64
}

// New creates an alarm clock with a maximum spin window of one minute.
// New는 바쁜대기 구간의 상한이 1분인 알람 시계를 생성합니다.
func New() *AlarmClock {
	return &AlarmClock{spinWindow: time.Minute}
}

// SetAlarm returns a shared channel that closes at or after the target time.
// SetAlarm은 목표 시각 이후 닫히는 공유 채널 하나를 반환합니다.
func (clock *AlarmClock) SetAlarm(at time.Time) <-chan struct{} {
	// Normalize only the map key; preserve the original clock basis for waiting.
	// 맵 키만 정규화하고 실제 대기에는 원래 시계 기준을 유지합니다.
	key := at.Round(0).UTC()

	clock.mu.Lock()
	defer clock.mu.Unlock()

	if clock.alarms == nil {
		clock.alarms = make(map[time.Time]*alarmGroup)
	}

	group := clock.alarms[key]
	if group == nil {
		window := clock.spinWindow
		var successor *alarmGroup
		var successorAt time.Time

		// Cap the new window by the predecessor gap and find the successor.
		// 직전 알람과의 간격으로 새 대기 구간을 제한하고 다음 알람을 찾습니다.
		for otherAt, other := range clock.alarms {
			if otherAt.Before(key) {
				window = min(window, key.Sub(otherAt))
			} else if successor == nil || otherAt.Before(successorAt) {
				successor, successorAt = other, otherAt
			}
		}

		group = &alarmGroup{}
		group.spinWindow.Store(int64(window))
		for i := range group.channels {
			group.channels[i] = make(chan struct{})
		}
		clock.alarms[key] = group

		// Inserting an alarm can only shorten the successor's spin window.
		// 알람을 삽입하면 다음 알람의 바쁜대기 구간은 짧아지기만 합니다.
		if successor != nil {
			window := min(
				time.Duration(successor.spinWindow.Load()),
				successorAt.Sub(key),
			)
			successor.spinWindow.Store(int64(window))
		}

		go clock.runAlarm(at, key, group)
	}

	ch := group.channels[group.next]
	group.next = (group.next + 1) % len(group.channels)
	return ch
}

func (clock *AlarmClock) runAlarm(at, key time.Time, group *alarmGroup) {
	var closed sync.WaitGroup
	closed.Add(len(group.channels))

	for _, ch := range group.channels {
		go func() {
			for {
				remaining := time.Until(at)
				if remaining <= 0 {
					break
				}

				// Recheck after sleeping and while spinning to observe a shorter window.
				// 잠에서 깬 뒤와 바쁜대기 중에 다시 확인해 줄어든 구간을 반영합니다.
				window := time.Duration(group.spinWindow.Load())
				if remaining > window {
					time.Sleep(remaining - window)
				}
			}

			close(ch)
			closed.Done()
		}()
	}

	closed.Wait()

	clock.mu.Lock()
	delete(clock.alarms, key)
	clock.mu.Unlock()
}
