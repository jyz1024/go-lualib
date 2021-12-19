# go-lualib
golang redis lub lib

# 使用方式
## 初始化
### 使用连接池
```go
   defer func() {
        if r := recover(); r != nil {
            log.Println("load lua lib panic:", r)
        }
    }()
    lualib.LoadLuaLibWithPool(poolImpl)
```
### 使用连接
```go
   defer func() {
        if r := recover(); r != nil {
            log.Println("load lua lib panic:", r)
        }
    }()
    lualib.LoadLuaLibWithConn(conn)
```
## 调用方式
1. 直接调用Eval执行脚本
2. RegisterScript + CallScript
3. 使用库本身提供的CompareAndSwap等方法

# redis.call(...argv)
- redis.call("Get","key"); boolean:false/string:"val" // tonumber(boolean) == nil 
- redis.call("IncrBy","key",1); number 