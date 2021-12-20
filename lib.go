package lualib

import (
	"errors"
	"log"
	"reflect"
	"strings"
	"sync"
)

type connPool interface {
	Get() Conn
}

type Conn interface {
	Do(string, ...interface{}) (interface{}, error)
	Close() error
}

type connDo interface {
	Do(string, ...interface{}) (interface{}, error)
}

var (
	ErrInsufficient = errors.New("error insufficient") // 扣减失败，余量不足
	ErrLockOccupied = errors.New("lock occupied")      // 锁被占用
	ErrValNotMatch  = errors.New("val not match")      // cas cad
)

// redigo lua lib
// inner script
const (
	ScriptCAS   = "CAS"   // compare and swap
	ScriptCAD   = "CAD"   // compare and delete
	ScriptHINC  = "HINC"  // hinc with check zero
	ScriptINC   = "INC"   // inc with check zero
	ScriptHMINC = "HMINC" // hminc with check zero

	scriptNum = 1 << 4
)

type libManager struct {
	pool      connPool
	conn      connDo
	scriptMap map[string]scriptInfo // script hash

	lock sync.RWMutex
}

type scriptInfo struct {
	KeyNumbers int
	Script     string
	Sha1       interface{} // []uint8
}

var libM *libManager

/********* init ********/
type innerConnPoolImpl struct {
	val reflect.Value // 原生val

	getMethod reflect.Value // val Get method
}

func (icpi innerConnPoolImpl) Get() Conn {
	callRes := icpi.getMethod.Call([]reflect.Value{})
	return callRes[0].Interface().(Conn)
}

func withInnerConnPool(val interface{}) connPool {
	refVal := reflect.ValueOf(val)
	methodVal := refVal.MethodByName("Get")
	if methodVal.IsZero() {
		panic("not impl Get method")
	}
	callRes := methodVal.Call([]reflect.Value{})
	if len(callRes) == 0 {
		panic("Get method return noting")
	}
	_, ok := callRes[0].Interface().(Conn)
	if !ok {
		panic("not impl Do method")
	}
	return innerConnPoolImpl{val: refVal, getMethod: methodVal}
}

// LoadLuaLibWithPool 使用连接池初始化lib
// pool需要实现 connPool 接口 或者提供Get方法，Get方法返回值实现Conn接口
func LoadLuaLibWithPool(pool interface{}) {
	libM = new(libManager)

	connPoolImpl, ok := pool.(connPool)
	if !ok {
		libM.pool = withInnerConnPool(pool)
	} else {
		libM.pool = connPoolImpl
	}

	libM.scriptMap = make(map[string]scriptInfo, scriptNum)
	loadInnerScript()
	log.Println("load lua lib with pool success")
}

// LoadLuaLibWithConn 使用连接初始化lib
// conn需要实现 connDo 接口
func LoadLuaLibWithConn(conn interface{}) {
	connImpl, ok := conn.(connDo)
	if !ok {
		panic("not implement do func")
	}
	libM.conn = connImpl
	loadInnerScript()
	log.Println("load lua lib with conn success")
}

/********* func ********/
func (lm *libManager) Do(cmdName string, args ...interface{}) (interface{}, error) {
	if lm.pool != nil {
		conn := lm.pool.Get()
		defer conn.Close()
		return conn.Do(cmdName, args...)
	}
	if lm.conn != nil {
		return lm.conn.Do(cmdName, args...)
	}
	return nil, errors.New("not init manager")
}

func loadInnerScript() {
	err := evalLoadScript(ScriptCAS, cmpAndSwapScript, 1)
	if err != nil {
		panic("load script error:" + ScriptCAS + ":" + err.Error())
	}
	err = evalLoadScript(ScriptCAD, cmpAndDelScript, 1)
	if err != nil {
		panic("load script error:" + ScriptCAD + ":" + err.Error())
	}
	err = evalLoadScript(ScriptHINC, hashHIncrByScript, 1)
	if err != nil {
		panic("load script error:" + ScriptHINC + ":" + err.Error())
	}
	err = evalLoadScript(ScriptINC, strIncrByScript, 1)
	if err != nil {
		panic("load script error:" + ScriptINC + ":" + err.Error())
	}
	err = evalLoadScript(ScriptHMINC, hmIncrByScript, 1)
	if err != nil {
		panic("load script error:" + ScriptHMINC + err.Error())
	}
}

// evalLoadScript 加载lua脚本到内存
func evalLoadScript(scriptKey string, script string, keyNumbers int) (err error) {
	libM.lock.Lock()
	defer libM.lock.Unlock()
	_, ok := libM.scriptMap[scriptKey]
	if ok {
		return errors.New("repeat load " + scriptKey)
	}
	scriptSha1, err := libM.Do("Script", "Load", script)
	if err != nil {
		return err
	}
	libM.scriptMap[scriptKey] = scriptInfo{Sha1: scriptSha1, KeyNumbers: keyNumbers, Script: script}
	return nil
}

// Exec 直接执行lua脚本
func Exec(script string, keyNumber int, argv ...interface{}) (interface{}, error) {
	return libM.Do("EVAL", append([]interface{}{script, keyNumber}, argv...)...)
}

// RegisterScript 注册脚本到Manager
func RegisterScript(scriptKey string, script string, keyNumbers int) error {
	return evalLoadScript(scriptKey, script, keyNumbers)
}

// CallScript 调用已注册的脚本
func CallScript(scriptKey string, argv ...interface{}) (interface{}, error) {
	libM.lock.RLock()
	script, ok := libM.scriptMap[scriptKey]
	if !ok {
		libM.lock.RUnlock()
		return nil, errors.New("script not loaded:" + scriptKey)
	}
	libM.lock.RUnlock()

	// 检查脚本是否存在，若不存在，重新load
	_, err := libM.Do("SCRIPT", " EXISTS", script.Sha1)
	if err != nil {
		DelScript(scriptKey)
		err = RegisterScript(scriptKey, script.Script, script.KeyNumbers)
		if err != nil {
			return nil, err
		}
	}

	args := make([]interface{}, 0, len(argv)+2)
	args = append(args, script.Sha1, script.KeyNumbers)
	args = append(args, argv...)
	return libM.Do("EVALSHA", args...)
}

// DelScript 仅从本地删除，并不会删除redis服务器缓存的lua脚本
func DelScript(scriptKey string) {
	libM.lock.Lock()
	defer libM.lock.Unlock()
	delete(libM.scriptMap, scriptKey)
}

func wrapErr(res interface{}, err error) (interface{}, error) {
	if err == nil {
		return res, err
	}
	if strings.Contains(err.Error(), ErrInsufficient.Error()) {
		return res, ErrInsufficient
	} else if strings.Contains(err.Error(), ErrValNotMatch.Error()) {
		return res, ErrValNotMatch
	} else if strings.Contains(err.Error(), ErrLockOccupied.Error()) {
		return res, ErrLockOccupied
	}
	return res, err
}

// CompareAndSwap 比较交换
// 如果对应key不存在，返回set结果
// 如果key存在，val 不等，返回err == ErrValNotMatch
// 如果key存在，val相等，返回set结果
func CompareAndSwap(key string, oriVal interface{}, tarVal interface{}) (err error) {
	_, err = wrapErr(CallScript(ScriptCAS, key, oriVal, tarVal))
	return
}

// CompareAndDel 比较删除
// key 不存在时，返回nil
// key存在且val相等时，返回del结果
// key存在切val不想等时，返回err == ErrValNotMatch
func CompareAndDel(key string, val interface{}) (err error) {
	_, err = wrapErr(CallScript(ScriptCAD, key, val))
	return
}

// HIncrBy 限制不能减到负值，不足时返回err == ErrInsufficient
func HIncrBy(key string, field string, increment interface{}) (interface{}, error) {
	return wrapErr(CallScript(ScriptHINC, key, field, increment))
}

// Inc 限制不能减到负值，不足时返回err == ErrInsufficient
func Inc(key string, increment interface{}) (interface{}, error) {
	return wrapErr(CallScript(ScriptINC, key, increment))
}

func HMIncrBy(key string, fields []interface{}, increments []interface{}) (interface{}, error) {
	if len(fields) != len(increments) {
		return nil, errors.New("field not match increment")
	}
	argv := make([]interface{}, 0, len(fields)*2+1)
	argv = append(argv, key)
	argv = append(argv, fields...)
	argv = append(argv, increments...)
	return wrapErr(CallScript(ScriptHMINC, argv...))
}

// Lock err == nil: lock success
// err == ErrLockOccupied 加锁失败
// redis error
func Lock(lockKey string, val interface{}, exSeconds int) error {
	res, err := libM.Do("Set", lockKey, val, "nx", "ex", exSeconds)
	if err != nil {
		return err
	}
	if res == nil {
		return ErrLockOccupied
	}
	return nil
}

// Unlock err == nil: unlock success
// err == ErrValNotMatch：val not match
// redis error
func Unlock(lockKey string, val interface{}) error {
	_, err := wrapErr(CallScript(ScriptCAD, lockKey, val))
	return err
}
