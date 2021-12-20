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
   ```go
   func main() {
      /******** get string key *********/
      singleValRes, err := lualib.Exec(`return redis.call("Get",KEYS[1])`, 1, "strkey")
       // set strkey 111
       // res type:[]uint8,err:<nil>
       log.Printf("res type:%T, res val:%v, err:%v\n", singleValRes, singleValRes, err)
   } 
   ```
2. RegisterScript + CallScript
    ```go
   func main() {
   	err := lualib.RegisterScript("GetVal", `return redis.call("Get",KEYS[1])`, 1)
       if err != nil {
           log.Println("register error:", err.Error())
       }
       res, err := lualib.CallScript("GetVal", "casKey")
       if err != nil {
           log.Println("err:", err.Error())
       } else {
           log.Println(res)
       }
      } 
   ```
3. 使用库本身提供的CompareAndSwap等方法
    ```go
   func main() {
       res, err := lualib.HMIncrBy("hashkey", []interface{}{"1", "2"}, []interface{}{10, -2})
       log.Printf("val:%v type:%T",res,res)
       if err != nil {
           if err == lualib.ErrInsufficient {
               log.Println("ErrInsufficient")
           } else {
               log.Println("incrby error:", err.Error())
           }
       } else {
           log.Println("incrby success:", res)
       }
   }
    ```
# redis.call(...argv)
- redis.call("Get","key"); boolean:false/string:"val" // tonumber(boolean) == nil 
- redis.call("IncrBy","key",1); number 