# alarmclock

Go로 구현한 바쁜대기(busy-wait) 알람입니다. `github.com/velar-agent/alarmclock`을 import해 사용합니다. `alarmclock.AlarmClock`은 지정한 시각까지 기다린 뒤, 기다리던 고루틴에서 콜백을 실행합니다.

## 실행

[go.mod](go.mod) 기준으로 Go 1.26.5 이상이 필요합니다. 저장소 루트에서 실행합니다.

```sh
go run ./cmd/compare
```

비교 프로그램은 바쁜대기 알람과 `time.After`를 시작 시각 다음 분 정각에 예약하고, 기다리는 동안 1초 간격의 카운터를 출력합니다. 예를 들어 07:32:27에 시작하면 목표는 07:33:00이며, 비교 결과는 07:33:01에 출력합니다. 출력 타이머도 스케줄링 지연의 영향을 받으므로 실제 출력 시각은 조금 늦어질 수 있습니다.

| 출력 항목 | 의미 |
|---|---|
| `Started at` | 프로그램 시작 시각 |
| `Alarm target` | 예약한 다음 분 정각 |
| `Report target` | 비교 결과를 출력할 목표 시각인 정분 1초 뒤 |
| `Counter` | 시작 이후 경과한 정수 초 |
| `Compared at` | 비교 결과 출력을 시작할 때 기록한 시각 |
| `Method` / `Fired at` | 대기 방식과 콜백 진입 또는 채널 수신 직후 기록한 시각 |
| `Lateness` | 기록한 실행 시각에서 목표 시각을 뺀 값 |
| `Difference` | `time.After`의 지연에서 바쁜대기 알람의 지연을 뺀 값 |
| `Environment` | OS·아키텍처·CPU 모델·논리 CPU 수·GOMAXPROCS·Go 버전·컴파일러 |

카운터는 진행 상황을 보여주는 용도입니다. 스케줄링과 출력에 시간이 걸릴 수 있으므로 매번 정확히 1초 간격으로 표시된다고 보장하지 않습니다.

환경 정보는 측정이 끝난 뒤 조회하며, 지원하지 않거나 조회에 실패한 항목은 `unavailable`로 표시합니다. 비교 결과는 같은 프로세스에서 수행한 단일 실행의 지연입니다.

## 패키지 사용

다음 예제는 2초 뒤 작업을 예약합니다. 콜백에서 실행 시각을 먼저 기록하고, 콜백이 끝난 뒤 출력합니다.

```go
package main

import (
	"fmt"
	"time"

	"github.com/velar-agent/alarmclock"
)

func main() {
	clock := alarmclock.New()
	at := time.Now().Add(2 * time.Second)

	var firedAt time.Time
	done := clock.SetAlarm(at, func() {
		// 출력 지연이 측정에 섞이지 않도록 실행 시각을 먼저 기록합니다.
		firedAt = time.Now()
	})

	// 콜백이 끝날 때까지 기다린 뒤 기록한 시각을 읽습니다.
	<-done
	fmt.Printf("Alarm fired: %s\n", firedAt.Format(time.RFC3339Nano))
	fmt.Printf("Lateness: %s\n", firedAt.Sub(at))
}
```

- `New`는 바쁜대기 구간을 1분으로 초기화한 알람 객체를 생성합니다.
- `SetAlarm`은 별도 고루틴에서 대기를 시작하고 완료 채널을 반환합니다.
- 콜백은 목표 시각에 도달한 것을 확인한 뒤, 기다리던 고루틴에서 실행합니다.
- 완료 채널은 콜백이 끝난 뒤 닫힙니다. 채널을 기다리는 시간에는 콜백의 실행 시간도 포함됩니다.

알람 구현은 [alarmclock.go](alarmclock.go), 분 정각 비교 프로그램은 [cmd/compare/main.go](cmd/compare/main.go), 환경 정보 조회는 [cmd/compare/environment.go](cmd/compare/environment.go)를 참고하시면 됩니다.

## 대기 방식과 CPU 비용

`New`로 생성한 객체는 목표까지 1분보다 많이 남았으면 먼저 초과 시간만큼 잠듭니다. 마지막 1분 동안은 현재 시각을 계속 확인하는 바쁜대기를 수행합니다. 목표까지 1분 이하로 남았다면 처음부터 바쁜대기를 시작합니다.

바쁜대기 구간에서는 CPU를 지속적으로 사용합니다. 분 정각 데모와 위의 2초 예약 예제는 대기 시간이 1분 이하이므로 처음부터 이 구간에 들어갑니다. 바쁜대기 구간은 `AlarmClock`의 내부 필드에 저장되며, `New`에서 1분으로 초기화합니다.

생성자를 거치지 않은 `AlarmClock`의 0값은 `spinWindow`가 0이므로, 목표 시각까지 먼저 잠든 뒤 남은 시간을 확인합니다. 1분 바쁜대기 구간을 사용하려면 `New`로 생성합니다.

## 정밀도 해석

- 분 정각 데모는 목표 시각과 경과 시간·지연 계산에 벽시계를 사용합니다. 시스템 시각이 조정되면 대기와 측정값에 영향을 줄 수 있습니다.
- `Fired at`은 콘솔에 표시된 순간이 아니라 콜백 진입 또는 채널 수신 직후 기록한 시각입니다. `time.After` 채널이 전달한 시각은 측정값으로 사용하지 않습니다. 출력은 완료 채널을 받은 뒤 수행합니다.
- `Lateness: 0s (0 ns)`는 이번 측정에서 기록한 실행 시각과 목표 시각의 값이 같았다는 뜻입니다. 실제 오차가 정확히 0이거나 이후 실행에서도 같은 결과가 나온다는 보장은 아닙니다.
- 나노초 자릿수까지 출력해도 시계가 1ns 단위로 갱신된다는 의미는 아닙니다. 검증한 macOS 환경의 벽시계 보고 해상도는 1µs였습니다.
- 바쁜대기는 운영체제와 Go 런타임의 스케줄링 지연을 없애지 못하므로, 고정된 최대 실행 지연을 보장하지 않습니다.

기존 실측과 조사 내용은 [측정 및 조사 기록](.agents/skills/verify-alarm-timing/references/observations.md)의 F01–F12에, 정분 비교 프로그램의 추가 실측과 검증 범위는 [2026-11-02 비교 실행 기록](.agents/skills/verify-alarm-timing/references/comparison-observations-2026-11-02.md)의 F13–F19에 정리되어 있습니다. 서로 다른 환경과 구현 단계에서 얻은 결과이므로, 각 기록의 조건을 확인하고 그 수치를 현재 구현의 성능 보장으로 해석하지 않습니다.
