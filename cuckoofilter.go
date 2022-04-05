package cuckoo

import (
	"fmt"
	"math/bits"
	"math/rand"
)

const maxCuckooCount = 500
const positiveCuckooCount = 3
const limitLoadFactor = 0.5
var redirect uint = 0

// Filter is a probabilistic counter
type Filter struct {
	buckets   []bucket
	count     uint
	bucketPow uint
	// 主动重定位
	pRedirect uint
}

// NewFilter returns a new cuckoofilter with a given capacity.
// A capacity of 1000000 is a normal default, which allocates
// about ~1MB on 64-bit machines.
func NewFilter(capacity uint) *Filter {
	capacity = getNextPow2(uint64(capacity)) / bucketSize
	if capacity == 0 {
		capacity = 1
	}
	buckets := make([]bucket, capacity)
	fmt.Println(capacity)
	return &Filter{
		buckets:   buckets,
		count:     0,
		bucketPow: uint(bits.TrailingZeros(capacity)),
	}
}

// Lookup returns true if data is in the counter
func (cf *Filter) Lookup(data []byte) bool {
	i1, fp := getIndexAndFingerprint(data, cf.bucketPow)
	if cf.buckets[i1].getFingerprintIndex(fp) > -1 {
		return true
	}
	i2 := getAltIndex(fp, i1, cf.bucketPow)
	return cf.buckets[i2].getFingerprintIndex(fp) > -1
}

// Reset ...
func (cf *Filter) Reset() {
	for i := range cf.buckets {
		cf.buckets[i].reset()
	}
	cf.count = 0
}

func randi(i1, i2 uint) uint {
	if rand.Intn(2) == 0 {
		return i1
	}
	return i2
}

//// Insert inserts data into the counter and returns true upon success
//func (cf *Filter) Insert(data []byte) bool {
//	// 第一步：通过元素值和哈希函数获取指纹值和第一个桶的位置
//	i1, fp := getIndexAndFingerprint(data, cf.bucketPow)
//	// 按照第一个求出的位置进行插入，成功直接返回，失败则继续
//	if cf.insert(fp, i1) {
//		return true
//	}
//	// 第二步，当第一个位置不为空时，根据第一步的指纹和位置进行异或操作得到第二个的位置
//	i2 := getAltIndex(fp, i1, cf.bucketPow)
//	// 按照第二个进行插入，成功直接返回，失败则继续
//	if cf.insert(fp, i2) {
//		return true
//	}
//	// 可以在此记录重定位操作
//	redirect++
//	return cf.reinsert(fp, randi(i1, i2))
//}

// Insert inserts data into the counter and returns true upon success
func (cf *Filter) Insert(data []byte) bool {
	// 第一步：通过元素值和哈希函数获取指纹值和第一个桶的位置
	i1, fp := getIndexAndFingerprint(data, cf.bucketPow)

	// 第二步，当第一个位置不为空时，根据第一步的指纹和位置进行异或操作得到第二个的位置
	i2 := getAltIndex(fp, i1, cf.bucketPow)

	// 第三步，选择两位置中负载少的桶进行插入
	if cf.buckets[i1].LoadFactor() < cf.buckets[i2].LoadFactor() {
		// 如果整体负载小于阈值，进行主动重定位式插入
		if cf.LoadFactor() < limitLoadFactor {
			// 如果候选桶的负载因子小于阈值直接插入
			if cf.buckets[i1].LoadFactor() < limitLoadFactor {
				// 直接插入
				if cf.insert(fp, i1) {
					return true
				}
			}
			// 如果候选桶的负载大于设置的阈值0.5，需要随机选取桶中的受害者,进行主动重定位的插入
			redirect++
			return cf.reinsert(fp,i1)
		}
		// 整体负载大于阈值，则进行整体重定位
		return cf.allReinsert(fp, i1)
	}
	// 如果整体负载小于阈值，进行主动重定位式插入
	if cf.LoadFactor() < limitLoadFactor {
		// 如果候选桶的负载因子小于阈值直接插入
		if cf.buckets[i2].LoadFactor() < limitLoadFactor {
			// 直接插入
			if cf.insert(fp, i2) {
				return true
			}
		}
		// 如果候选桶的负载大于设置的阈值0.5，需要随机选取桶中的受害者,进行主动重定位的插入
		redirect++
		return cf.reinsert(fp, i2)
	}
	// 整体负载大于阈值，则进行整体重定位
	return cf.reinsert(fp, i2)
}

// PositiveInsert 主动重定位插入函数
func (cf *Filter)PositiveInsert(fp fingerprint, i uint) bool {
	if cf.buckets[i].LoadFactor() < limitLoadFactor {
		// 按照第一个求出的位置进行插入，成功直接返回，失败则继续
		if cf.insert(fp, i) {
			return true
		}
	}
	// 如果候选桶的负载大于设置的阈值0.5，需要随机选取桶中的受害者,进行主动重定位的插入
	redirect++
	return cf.reinsert(fp,i)
}

// InsertUnique inserts data into the counter if not exists and returns true upon success
func (cf *Filter) InsertUnique(data []byte) bool {
	if cf.Lookup(data) {
		return false
	}
	return cf.Insert(data)
}

func (cf *Filter) insert(fp fingerprint, i uint) bool {
	if cf.buckets[i].insert(fp) {
		cf.count++
		return true
	}
	return false
}

//func (cf *Filter) reinsert(fp fingerprint, i uint) bool {
//	for k := 0; k < maxCuckooCount; k++ {
//		j := rand.Intn(bucketSize)
//		oldfp := fp
//		fp = cf.buckets[i][j]
//		cf.buckets[i][j] = oldfp
//
//		// look in the alternate location for that random element
//		i = getAltIndex(fp, i, cf.bucketPow)
//		if cf.insert(fp, i) {
//			return true
//		}
//	}
//	return false
//}

func (cf *Filter) reinsert(fp fingerprint, i uint) bool {
	var j int
	// 随机选取受害者
	if cf.buckets[i].LoadFactor() == 0.5 {
		j = rand.Intn(2)
	} else if cf.buckets[i].LoadFactor() == 0.75 {
		j = rand.Intn(3)
	} else if cf.buckets[i].LoadFactor() == 1 {
		j = rand.Intn(4)
	}
	// 插入新元素
	oldfp := fp
	fp = cf.buckets[i][j]
	cf.buckets[i][j] = oldfp
	cf.pRedirect++
	// 找到受害者元素的另一个位置
	// look in the alternate location for that random element
	i = getAltIndex(fp, i, cf.bucketPow)
	// 主动重定位次数小于阈值的话，继续进行主动重定位
	if cf.pRedirect < positiveCuckooCount {
		return cf.PositiveInsert(fp, i)
	}
	// 否则进行整体重定位
	return cf.allReinsert(fp,i)
}
// allReinsert 整体重定位函数
func (cf *Filter)allReinsert(fp fingerprint, i uint) bool {
	for k := 0; k < maxCuckooCount; k++ {
		if !cf.buckets[i].IsFull() {
			if cf.insert(fp, i) {
				return true
			}
		}
		j := rand.Intn(bucketSize)
		oldfp := fp
		fp = cf.buckets[i][j]
		cf.buckets[i][j] = oldfp
		cf.pRedirect++

		// look in the alternate location for that random element
		i = getAltIndex(fp, i, cf.bucketPow)
		if cf.insert(fp, i) {
			return true
		}
	}
	return false
}

// Delete data from counter if exists and return if deleted or not
func (cf *Filter) Delete(data []byte) bool {
	i1, fp := getIndexAndFingerprint(data, cf.bucketPow)
	if cf.delete(fp, i1) {
		return true
	}
	i2 := getAltIndex(fp, i1, cf.bucketPow)
	return cf.delete(fp, i2)
}

func (cf *Filter) delete(fp fingerprint, i uint) bool {
	if cf.buckets[i].delete(fp) {
		if cf.count > 0 {
			cf.count--
		}
		return true
	}
	return false
}

// Count returns the number of items in the counter
func (cf *Filter) Count() uint {
	return cf.count
}

func (cf *Filter)LoadFactor() float32 {
	return (float32(cf.count) / (float32(len(cf.buckets)) * bucketSize))
}

// Encode returns a byte slice representing a Cuckoofilter
func (cf *Filter) Encode() []byte {
	bytes := make([]byte, len(cf.buckets)*bucketSize)
	for i, b := range cf.buckets {
		for j, f := range b {
			index := (i * len(b)) + j
			bytes[index] = byte(f)
		}
	}
	return bytes
}

// Decode returns a Cuckoofilter from a byte slice
func Decode(bytes []byte) (*Filter, error) {
	var count uint
	if len(bytes)%bucketSize != 0 {
		return nil, fmt.Errorf("expected bytes to be multiple of %d, got %d", bucketSize, len(bytes))
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("bytes can not be empty")
	}
	buckets := make([]bucket, len(bytes)/4)
	for i, b := range buckets {
		for j := range b {
			index := (i * len(b)) + j
			if bytes[index] != 0 {
				buckets[i][j] = fingerprint(bytes[index])
				count++
			}
		}
	}
	return &Filter{
		buckets:   buckets,
		count:     count,
		bucketPow: uint(bits.TrailingZeros(uint(len(buckets)))),
	}, nil
}
