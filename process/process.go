package process

import (
	"fmt"
	"sync"
	"time"
)

// Message 프로세스 간의 메시지
type Message struct {
	From      int    // 메시지를 보낸 프로세스 ID
	To        int    // 메시지를 받는 프로세스 ID
	Vector    []int  // 메시지를 보낸 프로세스의 Vector Clock
	Event     string // 메시지 내용
	MessageID string // 메시지 고유 ID
	Timestamp int64  // 메시지 전송 시점
}

// VectorClockManager 모든 프로세스의 Vector Clock 관리
type VectorClockManager struct {
	Clock map[int][]int // 프로세스별 Vector Clock (프로세스 ID -> Vector Clock)
	Mu    sync.Mutex    // 동시성 제어
}

// Process 분산 시스템의 프로세스를 나타냄
type Process struct {
	ID        int                 // 프로세스 ID
	MessageCh chan Message        // 프로세스별 수신 채널
	ClockMgr  *VectorClockManager // Vector Clock 매니저
	Mu        sync.Mutex          // 동시성 제어
}

// NewVectorClockManager VectorClockManager 초기화
func NewVectorClockManager(n int) *VectorClockManager {
	clock := make(map[int][]int)
	for i := 0; i < n; i++ {
		clock[i] = make([]int, n) // 각 프로세스의 Vector Clock 초기화
	}
	return &VectorClockManager{Clock: clock}
}

// UpdateClock 특정 프로세스의 Vector Clock 업데이트
func (vcm *VectorClockManager) UpdateClock(processID int, receivedClock []int) {
	vcm.Mu.Lock()
	defer vcm.Mu.Unlock()

	if receivedClock != nil {
		// Vector Clocks merge: 최대값으로 병합
		for i := 0; i < len(receivedClock); i++ {
			if receivedClock[i] > vcm.Clock[processID][i] {
				vcm.Clock[processID][i] = receivedClock[i]
			}
		}
	}

	// 자신의 인덱스 값 증가 (로컬 이벤트 1 증가)
	vcm.Clock[processID][processID]++
}

// GetClock 특정 프로세스의 Vector Clock 반환
func (vcm *VectorClockManager) GetClock(processID int) []int {
	vcm.Mu.Lock()
	defer vcm.Mu.Unlock()

	// Vector Clock 복사본 반환
	clockCopy := make([]int, len(vcm.Clock[processID]))
	copy(clockCopy, vcm.Clock[processID])
	return clockCopy
}

// NewProcess Process 초기화
func NewProcess(id int, clockMgr *VectorClockManager) *Process {
	return &Process{
		ID:        id,
		MessageCh: make(chan Message, 1), // 프로세스별 채널 생성 (버퍼 크기 10)
		ClockMgr:  clockMgr,
	}
}

// SendMessage 메시지 전송 (상대 프로세스의 채널에 메시지를 보냄)
func (p *Process) SendMessage(to int, event string, targetCh chan<- Message, showDetails bool) {
	// (1) 송신 직전 로컬 시계 증가
	p.ClockMgr.UpdateClock(p.ID, nil)

	// (2) 현재 로컬 클럭 가져옴
	currentClock := p.ClockMgr.GetClock(p.ID)

	// (3) 메시지 생성
	msg := Message{
		From:      p.ID,
		To:        to,
		Vector:    currentClock,
		Event:     event,
		MessageID: fmt.Sprintf("%d-%d", p.ID, time.Now().UnixNano()),
		Timestamp: time.Now().Unix(),
	}

	// (4) 대상 프로세스의 채널로 전송
	targetCh <- msg

	// (5) 로그 출력
	if showDetails {
		fmt.Printf("Process %d: Sent message to Process %d: %v\n", p.ID, to, msg)
	} else {
		fmt.Printf("Process %d: Sent message to Process %d, Vector: %v\n", p.ID, to, msg.Vector)
	}
}

// ReceiveMessages 메시지 '한 번만' 수신
//
// 실제로는 무한 루프+고루틴 방식이 일반적이지만,
// "for 루프 구문 없이 단 한 번만" 메시지를 받도록 구성.
func (p *Process) ReceiveMessages(messageCh <-chan Message) {
	msg, ok := <-messageCh
	if !ok {
		fmt.Printf("Process %d: Channel closed\n", p.ID)
		return
	}
	p.Mu.Lock()

	// (1) 수신 메시지의 Clock 과 병합할 수 있으면 병합
	if p.CanMerge(msg.Vector) {
		p.ClockMgr.UpdateClock(p.ID, msg.Vector)
		fmt.Printf("Process %d: Received and merged message from %d, Vector: %v\n",
			p.ID, msg.From, p.ClockMgr.GetClock(p.ID))
	} else {
		fmt.Printf("Process %d: Received message from %d, Vector: %v\n",
			p.ID, msg.From, p.ClockMgr.GetClock(p.ID))
	}
	p.Mu.Unlock()
}

// CanMerge 메시지의 Vector Clock 과 현재 프로세스의 Vector Clock 병합 가능 여부
func (p *Process) CanMerge(receivedClock []int) bool {
	currentClock := p.ClockMgr.GetClock(p.ID)
	for i := 0; i < len(receivedClock); i++ {
		if receivedClock[i] > currentClock[i] {
			return true
		}
	}
	return false
}
