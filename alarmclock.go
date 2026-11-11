package alarmclock

import (
	"sync"
	"time"
)

const alarmChannelCount = 4

type AlarmClock struct {
	spinWindow time.Duration
	mu         sync.Mutex
	alarms     map[time.Time]*alarmGroup
}

type alarmGroup struct {
	channels [alarmChannelCount]chan struct{}
	next     int
}

// New creates an alarm clock with a one-minute spin window.
// New는 바쁜대기 구간이 1분인 알람 시계를 생성합니다.
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
		group = &alarmGroup{}
		for i := range group.channels {
			group.channels[i] = make(chan struct{})
		}
		clock.alarms[key] = group
		go clock.runAlarm(at, key, group)
	}

	ch := group.channels[group.next]
	group.next = (group.next + 1) % len(group.channels)
	return ch
}

func (clock *AlarmClock) runAlarm(at, key time.Time, group *alarmGroup) {
	var closed sync.WaitGroup
	closed.Add(len(group.channels))

	// Each goroutine waits independently and closes its own channel.
	// 각 고루틴이 독립적으로 기다린 뒤 자기 채널을 닫습니다.
	for _, ch := range group.channels {
		go func() {
			if remaining := time.Until(at); remaining > clock.spinWindow {
				time.Sleep(remaining - clock.spinWindow)
			}
			for time.Until(at) > 0 {
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
