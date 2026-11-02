package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const unavailable = "unavailable"

func printEnvironment() {
	fmt.Println("\nEnvironment:")
	fmt.Printf("OS / Arch:   %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("OS version:  %s\n", osVersion())
	fmt.Printf("CPU model:   %s\n", cpuModel())
	// NumCPU는 현재 프로세스에서 사용할 수 있는 논리 CPU 수입니다.
	fmt.Printf("Logical CPUs: %d\n", runtime.NumCPU())
	fmt.Printf("GOMAXPROCS:  %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("Go version:  %s\n", runtime.Version())
	fmt.Printf("Compiler:    %s\n", runtime.Compiler)
}

func osVersion() string {
	switch runtime.GOOS {
	case "darwin":
		return fmt.Sprintf("macOS %s (build %s)",
			systemOutput("sw_vers", "-productVersion"),
			systemOutput("sw_vers", "-buildVersion"))
	case "linux":
		return systemOutput("uname", "-sr")
	case "windows":
		return systemOutput("cmd", "/c", "ver")
	default:
		return unavailable
	}
}

func cpuModel() string {
	switch runtime.GOOS {
	case "darwin":
		return systemOutput("sysctl", "-n", "machdep.cpu.brand_string")
	case "linux":
		data, err := os.ReadFile("/proc/cpuinfo")
		if err != nil {
			return unavailable
		}
		// CPU 아키텍처마다 모델명을 나타내는 필드가 다를 수 있습니다.
		lines := strings.Split(string(data), "\n")
		for _, field := range []string{"model name", "Hardware", "Processor"} {
			for _, line := range lines {
				key, value, ok := strings.Cut(line, ":")
				if ok && strings.TrimSpace(key) == field {
					if value = strings.TrimSpace(value); value != "" {
						return value
					}
				}
			}
		}
	case "windows":
		return systemOutput("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
			"(Get-CimInstance Win32_Processor | Select-Object -First 1).Name")
	}
	return unavailable
}

func systemOutput(name string, args ...string) string {
	// 환경 조회 명령이 지연되어도 프로그램이 계속 기다리지 않도록 제한합니다.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = 100 * time.Millisecond
	output, err := cmd.Output()
	if err != nil {
		return unavailable
	}
	if value := strings.TrimSpace(string(output)); value != "" {
		return value
	}
	return unavailable
}
