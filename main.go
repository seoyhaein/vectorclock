package main

import (
	"log"
	"time"

	vc "github.com/seoyhaein/vectorclock/process"
)

// main 예제 실행
func main() {
	n := 3                                  // 프로세스 수
	clockMgr := vc.NewVectorClockManager(n) // Vector Clock 매니저 생성

	// 프로세스 초기화 (각 프로세스가 자기 채널 보유)
	processes := make([]*vc.Process, n)
	for i := 0; i < n; i++ {
		processes[i] = vc.NewProcess(i, clockMgr)
	}

	/*
	   예시 시나리오
	   (1) P0 -> P1 메시지 전송, P1이 한 번만 Receive
	   (2) P1 -> P0 메시지 전송, P0이 한 번만 Receive
	*/

	// (1) P0 -> P1 전송, P1 수신
	processes[0].SendMessage(1, "Message from P0 to P1", processes[1].MessageCh, false)
	processes[1].ReceiveMessages(processes[1].MessageCh)

	// (2) P1 -> P0 전송, P0 수신
	processes[1].SendMessage(0, "Message from P1 to P0", processes[0].MessageCh, false)
	processes[0].ReceiveMessages(processes[0].MessageCh)

	// 잠시 대기 (2초) 후 프로그램 종료
	time.Sleep(2 * time.Second)

	// 모든 채널 닫기 (한 번만 수신한다면 사실상 큰 의미는 없지만, 정리 차원)
	for i := 0; i < n; i++ {
		close(processes[i].MessageCh)
	}

	log.Println("Simulation stopped gracefully.")
}
