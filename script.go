package lualib

// compare and swap
const cmpAndSwapScript = `	
	local getVal = redis.call("Get",KEYS[1])
	if getVal == false then 
		return redis.call("Set",KEYS[1],ARGV[2])
	end
	if getVal == ARGV[1] then 
		return redis.call("Set",KEYS[1],ARGV[2])
	else
		return error("val not match",0)
	end
`

// compare and delete
const cmpAndDelScript = `
	local getVal = redis.call("Get",KEYS[1])
	if getVal == false then 
		return 0
	end
	if getVal == ARGV[1] then
		return redis.call("Del", KEYS[1])
	else
		return error("val not match",0)
	end
`

// hash reduce with check
const hashHIncrByScript = `
	local increment = tonumber(ARGV[2])
	assert(type(increment)=="number","increment type error")
	if increment >= 0 then 
		return redis.call("HIncrBy",KEYS[1],ARGV[1],ARGV[2])
	else
		local getVal = tonumber(redis.call("HGet",KEYS[1],ARGV[1]))
		assert(type(getVal)=="number", "error insufficient")
		if getVal < -increment then 
			return error("error insufficient",0)
		else
			return redis.call("HIncrBy",KEYS[1],ARGV[1],ARGV[2])
		end
	end
`

// str reduce with check
const strIncrByScript = `
	local increment = tonumber(ARGV[1])
	assert(type(increment)=="number","increment type error")
	if increment >= 0 then 
		return redis.call("IncrBy",KEYS[1],ARGV[1])
	else
		local getVal = tonumber(redis.call("Get",KEYS[1]))
		assert(type(getVal)=="number", "error insufficient")
		if getVal < -increment then 
			return error("error insufficient",0)
		else
			return redis.call("IncrBy",KEYS[1],ARGV[1])
		end
	end
`

const hmIncrByScript = `
	local fieldsNum = #ARGV
	if fieldsNum == 0 then 
		return {}
	end
	local argvTable = {}
	table.insert(argvTable,"HMGet")
	table.insert(argvTable,KEYS[1])
	for k,v in pairs(ARGV) do 
		if k > fieldsNum / 2 then
			break
		end
		table.insert(argvTable,v)
	end
	local getValList = redis.call(unpack(argvTable))
	assert(type(getValList)=="table","error insufficient")
	local index = 1
	local fieldsTable = {}
	local resTable = {}
	table.insert(fieldsTable,"HMSet")
	table.insert(fieldsTable,KEYS[1])
	for i=fieldsNum/2+1,fieldsNum,1 do
		local increment = tonumber(ARGV[i])
		assert(type(increment)=="number","increment type error")
		local getVal = tonumber(getValList[index])
		assert(type(getVal)=="number","error insufficient")
		if increment < 0 then 
			if getVal + increment < 0 then 
				return error("error insufficient",0)
			end
		end
		table.insert(fieldsTable,ARGV[index])
		table.insert(fieldsTable,getVal+increment)
		table.insert(resTable,getVal+increment)
		index = index + 1
	end
	local hmSetRes = redis.call(unpack(fieldsTable))
	assert(type(hmSetRes)=="table","incr error")
	return resTable
`
