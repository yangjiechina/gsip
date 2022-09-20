package sip

import "time"

func min(i1, i2 int) int {
	if i1 < i2 {
		return i1
	} else {
		return i2
	}
}

type Timer interface {
	start(task func() bool)

	stop()

	reset()
}

type timer struct {
	t        *time.Timer
	interval int
}

func (t *timer) start(task func()) {
	t.t = time.AfterFunc(time.Duration(t.interval)*time.Millisecond, task)
}

func (t *timer) stop() {
	t.t.Stop()
}

func (t *timer) reset() {
	t.t.Reset(time.Duration(t.interval) * time.Millisecond)
}

type timerA struct {
	timer
}

func (t *timerA) start(task func() bool) {
	t.interval = T1
	t.timer.start(func() {
		if !task() {
			t.interval = t.interval * 2
			t.timer.reset()
		}
	})
}

type timerB struct {
	timer
}

func (t *timerB) start(task func()) {
	t.interval = 64 * T1
	t.timer.start(func() {
		task()
	})
}

type timerD struct {
	timer
}

func (t *timerD) start(task func()) {
	t.interval = 32 * 1000
	t.timer.start(func() {
		task()
	})
}

type timerE struct {
	timer
}

func (t *timerE) start(task func() bool) {
	t.interval = T1
	t.timer.start(func() {
		if !task() {
			if t.interval != T2 {
				t.interval = min(t.interval*2, T2)
			}
			t.timer.reset()
		}
	})
}

func (t *timerE) setToT2() {
	t.interval = T2
}

type timerF struct {
	timer
}

func (t *timerF) start(task func()) {
	t.interval = 64 * T1
	t.timer.start(func() {
		task()
	})
}

type timerK struct {
	timer
}

func (t *timerK) start(task func()) {
	t.interval = T4
	t.timer.start(func() {
		task()
	})
}

//-----------------------------------//
//G H I

type timerG struct {
	timerE
}

type timerH struct {
	timerF
}

type timerI struct {
	timerK
}

type timerJ struct {
	timerF
}
