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
