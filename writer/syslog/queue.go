/**
**队列模块。使用channel作为存放数据的存储
 */
package syslog

import (
	"time"
)

type queue struct {
	value   chan interface{}
	maxSize int
	timeout time.Duration
	isClose bool
}

//func (q *queue) Get() (interface{}, bool) {
//	if !q.isClose {
//		for i := 0; i < 3; i++ {
//			select {
//			case v, ok := <-q.value:
//				if !ok {
//					return nil, false
//				}
//				return v, true
//			default:
//				time.Sleep(q.timeout)
//			}
//			//case <-time.After(q.timeout):
//			//	return nil, false
//			//}
//		}
//	}
//	return nil, false
//}
func (q *queue) Get() (interface{}, bool) {
	if !q.isClose {
		t := GetTimer()
		t.Reset(q.timeout)
		select {
		case v, ok := <-q.value:
			if !ok {
				PutTimer(t)
				return nil, false
			}
			PutTimer(t)
			return v, true
		case <-t.C:
			PutTimer(t)
			return nil, false
		}
	}
	return nil, false
}

//func (q *queue) Put(v interface{}) bool {
//	if !q.isClose {
//		for i := 0; i < 3; i++ {
//			select {
//			case q.value <- v:
//				return true
//			default:
//				time.Sleep(q.timeout)
//			}
//		}
//
//	}
//	return false
//}
func (q *queue) Put(v interface{}) bool {
	if !q.isClose {
		t := GetTimer()
		t.Reset(q.timeout)
		select {
		case q.value <- v:
			PutTimer(t)
			return true
		case <-t.C:
			PutTimer(t)
			return false
		}
	}
	return false
}

func (q *queue) Size() int {
	return len(q.value)
}

func (q *queue) Empty() bool {
	return len(q.value) == 0
}

func (q *queue) Full() bool {
	return len(q.value) == cap(q.value)
}

func (q *queue) Close() {
	if !q.isClose {
		q.isClose = true
		close(q.value)
	}
}

func NewQueue(maxSize int, timeout time.Duration) *queue {
	queue := queue{
		value:   make(chan interface{}, maxSize),
		maxSize: maxSize,
		timeout: timeout, //timeout用来指定读取缓存的超时时间，超过这个时间就读取失败
	}
	return &queue
}
