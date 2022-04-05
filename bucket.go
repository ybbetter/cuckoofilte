package cuckoo

type fingerprint byte

type bucket [bucketSize]fingerprint

const (
	nullFp     = 0
	bucketSize = 4
)

func (b *bucket) insert(fp fingerprint) bool {
	for i, tfp := range b {
		if tfp == nullFp {
			b[i] = fp
			return true
		}
	}
	return false
}

func (b *bucket)IsFull() bool {
	for _, tfp := range b {
		if tfp == nullFp {
			return false
		}
	}
	return true
}

func (b *bucket)LoadFactor() float32 {
	count := 0
	for _, tmp := range b {
		if tmp != nullFp {
			count++
		}
	}
	return float32(count) / float32(bucketSize)
}

func (b *bucket) delete(fp fingerprint) bool {
	for i, tfp := range b {
		if tfp == fp {
			b[i] = nullFp
			return true
		}
	}
	return false
}

func (b *bucket) getFingerprintIndex(fp fingerprint) int {
	for i, tfp := range b {
		if tfp == fp {
			return i
		}
	}
	return -1
}

func (b *bucket) reset() {
	for i := range b {
		b[i] = nullFp
	}
}
