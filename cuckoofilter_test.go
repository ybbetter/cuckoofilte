package cuckoo

import (
	"bufio"
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"gonum.org/v1/plot/vg"
	"io"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
)

func TestInsertion(t *testing.T) {
	cf := NewFilter(1000000)
	fd, err := os.Open("/usr/share/dict/words")
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(fd)

	var values [][]byte
	var lineCount uint
	for scanner.Scan() {
		s := []byte(scanner.Text())
		if cf.InsertUnique(s) {
			lineCount++
		}
		values = append(values, s)
	}

	count := cf.Count()
	if count != lineCount {
		t.Errorf("Expected count = %d, instead count = %d", lineCount, count)
	}

	for _, v := range values {
		cf.Delete(v)
	}

	count = cf.Count()
	if count != 0 {
		t.Errorf("Expected count = 0, instead count == %d", count)
	}
}

func TestEncodeDecode(t *testing.T) {
	cf := NewFilter(8)
	cf.buckets = []bucket{
		[4]fingerprint{1, 2, 3, 4},
		[4]fingerprint{5, 6, 7, 8},
	}
	cf.count = 8
	bytes := cf.Encode()
	ncf, err := Decode(bytes)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !reflect.DeepEqual(cf, ncf) {
		t.Errorf("Expected %v, got %v", cf, ncf)
	}
}

func TestDecode(t *testing.T) {
	ncf, err := Decode([]byte(""))
	if err == nil {
		t.Errorf("Expected err, got nil")
	}
	if ncf != nil {
		t.Errorf("Expected nil, got %v", ncf)
	}
}

func BenchmarkFilter_Reset(b *testing.B) {
	const cap = 10000
	filter := NewFilter(cap)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.Reset()
	}
}

func CuckooPlot(v interface{}, load []float32,n int)  {
	p := plot.New()
	p.Title.Text= "Started"
	p.X.Label.Text = "Factor"
	p.Y.Label.Text = "Redirect"
	err := plotutil.AddLinePoints(p,"Line", CreatePoint(v,load,n))
	if err !=nil {
		fmt.Println(err)
	}
	if _, ok := v.([]uint); ok {
		if err = p.Save(4*vg.Inch, 4*vg.Inch, "points_redirect.png"); err != nil {
			fmt.Println(err)
		}
	} else if _, ok := v.([]time.Duration); ok {
		if err = p.Save(4*vg.Inch, 4*vg.Inch, "points_time.png"); err != nil {
			fmt.Println(err)
		}
	}

}

func CreatePoint(v interface{}, load []float32,n int) plotter.XYs {
	points := make(plotter.XYs, n)
	if res, ok := v.([]uint); ok {
		for i := range points {
			points[i].X = float64(load[i])
			points[i].Y = float64(res[i])
		}
	} else if res, ok := v.([]time.Duration); ok{
		for i := range points {
			points[i].X = float64(load[i])
			points[i].Y = float64(res[i].Nanoseconds())/60000000
		}
	}

	return points
}

func TestFilter_Insert_Time(t *testing.T) {
	const cap = 100000
	filter := NewFilter(cap)

	//b.ResetTimer()
	//var redirectList []uint
	var factors []float32
	var hash [32]byte

	var timeList []int64
	// 124518
	// 131072
	start := time.Now()
	//fmt.Println(start)
	for i := 0; i < 131072; i++ {

		io.ReadFull(rand.Reader, hash[:])
		filter.Insert(hash[:])
		//elapsed := time.Since(start)
		elapsed := time.Since(start)
		timeList = append(timeList, elapsed.Nanoseconds())
		//fmt.Println("时延: ", elapsed)
		//redirectList = append(redirectList, redirect)
		factors = append(factors, filter.LoadFactor())
		//fmt.Println("重定位次数: ",redirect)
		//fmt.Println("负载因子: ",filter.LoadFactor())
	}
	for i := 0; i < 131072; i++ {
		fmt.Println("负载因子: ",factors[i])
		fmt.Println(timeList[i])
	}
	filename := "data-time-modify.csv"
	File,err:=os.OpenFile(filename,os.O_RDWR|os.O_APPEND|os.O_CREATE,0666)
	if err!=nil{
		fmt.Println("文件打开失败！")
	}
	defer File.Close()
	writer := csv.NewWriter(File)
	for i := 0; i < 131072; i++ {
		insertData := []string{strconv.FormatInt(timeList[i],10),strconv.FormatFloat(float64(factors[i]),'f',10,32)}
		err = writer.Write(insertData)
		if err != nil {
			fmt.Println("写入文件失败")
		}
		writer.Flush()
	}
	////CuckooPlot(redirectList,factors,100000)
	////CuckooPlot(timeList,factors,100000)
}

func TestFilter_Insert_FullBucket(t *testing.T) {
	const cap = 100000
	filter := NewFilter(cap)

	//b.ResetTimer()
	//var redirectList []uint
	var factors []float32
	var fullbuckets []uint
	var fullbucket uint
	var hash [32]byte

	//var timeList []time.Duration
	// 124518
	// 131072
	//start := time.Now()
	for i := 0; i < 124518; i++ {
		fullbucket = 0
		io.ReadFull(rand.Reader, hash[:])
		filter.Insert(hash[:])
		for _, bucket := range filter.buckets {
			if bucket.IsFull() {
				fullbucket++
			}
		}
		fullbuckets = append(fullbuckets, fullbucket)
		//elapsed := time.Since(start)
		//timeList = append(timeList, elapsed)
		//fmt.Println("时延: ", elapsed)
		//redirectList = append(redirectList, redirect)
		factors = append(factors, filter.LoadFactor())
		//fmt.Println("重定位次数: ",redirect)
		//fmt.Println("负载因子: ",filter.LoadFactor())
	}
	for i := 0; i < 124518; i++ {
		fmt.Println("负载因子: ",factors[i])
		fmt.Println("满桶个数: ",fullbuckets[i])
	}
	filename := "data-bucket-old.csv"
	File,err:=os.OpenFile(filename,os.O_RDWR|os.O_APPEND|os.O_CREATE,0666)
	if err!=nil{
		fmt.Println("文件打开失败！")
	}
	defer File.Close()
	writer := csv.NewWriter(File)
	for i := 0; i < 124518; i++ {
		insertData := []string{strconv.FormatUint(uint64(fullbuckets[i]),10),strconv.FormatFloat(float64(factors[i]),'f',10,32)}
		err = writer.Write(insertData)
		if err != nil {
			fmt.Println("写入文件失败")
		}
		writer.Flush()
	}
	////CuckooPlot(redirectList,factors,100000)
	////CuckooPlot(timeList,factors,100000)
}

func TestFilter_Insert(t *testing.T) {
	const cap = 100000
	filter := NewFilter(cap)

	//b.ResetTimer()
	var redirectList []uint
	var factors []float32
	var hash [32]byte
	start := time.Now()
	//var timeList []time.Duration
	// 124518
	// 131072
	for i := 0; i < 124518; i++ {

		io.ReadFull(rand.Reader, hash[:])
		filter.Insert(hash[:])

		//timeList = append(timeList, elapsed)
		//fmt.Println("时延: ", elapsed.Nanoseconds())
		redirectList = append(redirectList, redirect)
		factors = append(factors, filter.LoadFactor())
		//fmt.Println("重定位次数: ",redirect)
		//fmt.Println("负载因子: ",filter.LoadFactor())
	}
	elapsed := time.Since(start)
	fmt.Println("time elapse in nano: ", elapsed)
	filename := "data-old2.csv"
	File,err:=os.OpenFile(filename,os.O_RDWR|os.O_APPEND|os.O_CREATE,0666)
	if err!=nil{
		fmt.Println("文件打开失败！")
	}
	defer File.Close()
	writer := csv.NewWriter(File)
	for i := 0; i < 124518; i++ {
		insertData := []string{strconv.FormatUint(uint64(redirectList[i]),10),strconv.FormatFloat(float64(factors[i]),'f',10,32)}
		err = writer.Write(insertData)
		if err != nil {
			fmt.Println("写入文件失败")
		}
		writer.Flush()
	}
	//CuckooPlot(redirectList,factors,100000)
	//CuckooPlot(timeList,factors,100000)
}

func BenchmarkFilter_Insert(b *testing.B) {
	const cap = 100000
	filter := NewFilter(cap)

	//b.ResetTimer()
	var redirectList []uint
	var factors []float32
	var hash [32]byte
	start := time.Now()
	//var timeList []time.Duration
	// 124518
	// 131072
	for i := 0; i < 124518; i++ {

		io.ReadFull(rand.Reader, hash[:])
		filter.InsertUnique(hash[:])

		//timeList = append(timeList, elapsed)
		//fmt.Println("时延: ", elapsed.Nanoseconds())
		redirectList = append(redirectList, redirect)
		factors = append(factors, filter.LoadFactor())
		//fmt.Println("重定位次数: ",redirect)
		//fmt.Println("负载因子: ",filter.LoadFactor())
	}
	elapsed := time.Since(start)
	fmt.Println("time elapse in nano: ", elapsed)
	filename := "data-old2.csv"
	File,err:=os.OpenFile(filename,os.O_RDWR|os.O_APPEND|os.O_CREATE,0666)
	if err!=nil{
		fmt.Println("文件打开失败！")
	}
	defer File.Close()
	writer := csv.NewWriter(File)
	for i := 0; i < 124518; i++ {
		insertData := []string{strconv.FormatUint(uint64(redirectList[i]),10),strconv.FormatFloat(float64(factors[i]),'f',10,32)}
		err = writer.Write(insertData)
		if err != nil {
			fmt.Println("写入文件失败")
		}
		writer.Flush()
	}
	//CuckooPlot(redirectList,factors,100000)
	//CuckooPlot(timeList,factors,100000)
}

func BenchmarkFilter_Lookup(b *testing.B) {
	const cap = 10000
	filter := NewFilter(cap)

	var hash [32]byte
	for i := 0; i < 10000; i++ {
		io.ReadFull(rand.Reader, hash[:])
		filter.Insert(hash[:])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		io.ReadFull(rand.Reader, hash[:])
		filter.Lookup(hash[:])
	}
}
