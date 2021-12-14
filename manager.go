package log

import (
    "errors"
    "sync/atomic"
)

var (
    manager   atomic.Value

    ErrExistedLogger = errors.New("existed logger")
    ErrEmptyLoggerID = errors.New("empty logger ID")
)

// GetLogger 获取全局 logger
func GetLogger(logID string)(*Logger,bool){
    i := manager.Load()
    c, ok := i.(map[string]*Logger)
    if !ok{
        return nil,ok
    }
    l,ok := c[logID]
    return l,ok
}

func registerLogger(l *Logger,force bool)error{
    if l.ID == ""{
        return ErrEmptyLoggerID
    }
    i := manager.Load()
    oldManager, ok := i.(map[string]*Logger)
    newManager := make(map[string]*Logger,len(oldManager)+1)
    if ok{
        for k,v := range oldManager{
            newManager[k]=v
            delete(oldManager,k)
        }
    }

    l2,existed := newManager[l.ID]

    if existed && !force && l2 != l{
       return ErrExistedLogger
    }
    newManager[l.ID] = l
    // 如果是 root logger，则默认更新全局log
    if l.ID == DefaultName{
        UpdateGlobalLog(l)
    }
    manager.Store(newManager)
    return nil
}


// RegisterLogger 注册 logger 到全局管理器中，方便通过 GetLogger 获取logger
// 请勿并发操作该函数，避免 manager 出现logger对象存储失败
func RegisterLogger(l *Logger)(error){
    return registerLogger(l,false)
}

// ReplaceLogger 替换 全局管理器中的logger 对象
// 请勿并发操作该函数，避免 manager 出现logger对象存储失败
func ReplaceLogger(l *Logger)error{
    return registerLogger(l,true)
}


// DeRegisterLogger 注销 全局管理器中 的 logger
// 请勿并发操作该函数，避免 manager 出现logger对象存储失败
func DeRegisterLogger(l *Logger){
    i := manager.Load()

    oldManager, ok := i.(map[string]*Logger)
    newManager := make(map[string]*Logger,len(oldManager)+1)
    if ok{
        for k,v := range oldManager{
            if k != l.ID{
                newManager[k]=v
            }
            delete(oldManager,k)
        }
    }
    manager.Store(newManager)
}
