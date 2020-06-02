package gorc

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
)

func Download(url string, thread int64, manual bool, root string, blockSize int64, attempt int, filename string) (err error) {
	//flag.Parse()
	var group sync.WaitGroup
	Context := assign(url, thread, manual, root, blockSize, attempt, filename)
	if Context == nil {
		return errors.New("not support")
	}
	go removeCache(Context)
	log.Println("start download")
	previous := time.Now()

	for key, meta := range Context.fileNames {
		if checkBlockStat(key, meta) {
			continue
		}
		log.Println("file", key, "start", meta.end-meta.start+1)
		group.Add(1)
		go func(pi chan string, url string, address string, b *block, attempt int) {
			defer group.Done()
			goBT(pi, url, address, b, attempt)
		}(Context.Pi, Context.file.url, key, meta, int(Context.Attempt))
	}
	time.Sleep(2 * time.Second)
	goBar(Context, Context.file.length, previous)
	group.Wait()
	//log.Println("start unzip")
	err = createFileOnly(Context.file.filePath)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	for i := len(Context.tempList) - 1; i >= 0; i-- {
		err = appendToFile(Context.file.filePath, string(readFile(Context.tempList[i])))
		if err != nil {
			log.Println(err.Error(), "download request failed,please retry")
			return
		}
		if i == 0 {
			Context.Exit <- true
		}
	}

	flag := <-Context.Exit
	if flag {
		for _, file := range Context.tempList {
			deleteFile(file)
		}
		log.Println("download completed")
		return
	}
	log.Println("download request failed,please retry")
	return
}
func goBT(pi chan string, url string, address string, b *block, attempt int) {
	l, err := sendGet(url, address, b.start, b.end)
	if err != nil || l != (b.end-b.start+1) {
		log.Println("下载重试中")
		if b.count > attempt {
			pi <- b.id
			err = nil
		}
		if b.count <= attempt {
			b.count++
			goBT(pi, url, address, b, attempt)
		}
	}
	return
}
func removeCache(context2 *context) {
	for {
		select {
		case str := <-context2.Pi:
			p := filePath(context2.TmpPath, str)
			deleteFile(p)
			context2.Exit <- false
			context2.ExitSub <- false
		case <-context2.ExitSub:
			break
		}
	}
}
func bar(count, size int) string {
	str := ""
	for i := 0; i < size; i++ {
		if i < count {
			str += "="
		} else {
			str += " "
		}
	}
	return str
}

func goBar(Context *context, length int64, t time.Time) {
	for {
		var sum int64 = 0
		for key, _ := range Context.fileNames {
			sum += getFileSize(key)
		}
		percent := getPercent(sum, length)
		result, _ := strconv.Atoi(percent)
		str := "working " + percent + "%" + "[" + bar(result, 100) + "] " + " " + fmt.Sprintf("%.f", getCurrentSize(t)) + "s"
		fmt.Printf("\r%s", str)
		time.Sleep(1 * time.Second)
		if sum == length {
			fmt.Println("")
			break
		}
	}
}
func getPercent(a int64, b int64) string {
	result := float64(a) / float64(b) * 100
	return fmt.Sprintf("%.f", result)
}
func getCurrentSize(t time.Time) float64 {
	return time.Now().Sub(t).Seconds()
}
