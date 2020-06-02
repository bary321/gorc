package gorc

import (
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	LEVEL   int64 = 1024
	RULE    int64 = LEVEL * LEVEL * 16
	WINDOWS       = "windows"
	LINUX         = "linux"
)

type File struct {
	url      string
	name     string
	length   int64
	filePath string
}
type block struct {
	previous *block
	id       string
	start    int64
	end      int64
	count    int
}
type context struct {
	fileNames map[string]*block
	file      *File
	tempList  []string
	Attempt   int
	BlockSize int64
	TmpPath   string
	Exit      chan bool
	ExitSub   chan bool
	Pi        chan string
}

//var Context *context = new(context)
var Count int

func assign(url string, thread int64, manual bool, tmpPath string, blockSize int64, attempt int, fileName string) *context {
	var group sync.WaitGroup
	Context := new(context)
	Context.BlockSize = blockSize
	Context.Attempt = attempt
	Context.TmpPath = tmpPath
	Context.Pi = make(chan string, 2)
	Context.Exit = make(chan bool, 1)
	Context.ExitSub = make(chan bool, 1)

	tName, fName := searchName(url)
	if len(fileName) > 0 {
		fName = fileName
	}
	length, agree, err := sendHead(url)
	if err != nil {
		log.Println("get file length failed")
		return nil
	}
	l, _ := strconv.ParseInt(length, 10, 64)
	if !checkFileStat(tmpPath) {
		err := os.MkdirAll(tmpPath, 0666)
		if err != nil {
			panic(err)
		}
	}
	if !agree {
		log.Println("资源不支持断点续传模式,单线程模式执行中")
		addr := filePath(tmpPath, fName)
		group.Add(1)
		go func() {
			defer group.Done()
			singleThread(Context, url, addr, l)
		}()
		time.Sleep(3 * time.Second)
		ps := time.Now()
		goBar(Context, l, ps)
		group.Wait()
		log.Println("download completed")
		return nil
	}
	f := &File{url: url, name: fName, length: l, filePath: filePath(tmpPath, fName)}
	Context.file = f
	var element *block
	if manual {
		log.Println("manual pattern")
		element = partFileManual(l, thread, tName)
		Context = assignBlock(Context, element)
		return Context
	}
	if l <= (RULE * blockSize) {
		log.Println("default manual pattern")
		element = partFileManual(l, thread, tName)
		Context = assignBlock(Context, element)
		return Context
	}
	log.Println("auto pattern")
	element = partFile(blockSize, l, 0, l-1)
	Context = assignBlock(Context, element)
	return Context
}

func singleThread(Context *context, url string, address string, length int64) {
	k := new(block)
	k.start = 0
	k.end = length - 1
	k.id = address
	m := make(map[string]*block)
	m[address] = k
	Context.fileNames = m
	goBT(Context.Pi, url, address, k, int(Context.Attempt))
}

func searchName(url string) (tmpName, fullName string) {
	u := []byte(url)
	s := strings.LastIndex(url, "/")
	if s == -1 {
		s = 0
		fullName = string(u[s:])
	} else {
		fullName = string(u[s+1:])
	}
	t := []byte(fullName)
	d := strings.LastIndex(fullName, ".")
	if d == -1 {
		d = len(t)
		tmpName = string(t[:])
	} else {
		tmpName = string(t[:d])
	}
	return
}

func assignBlock(Context *context, b *block) *context {
	if b == nil {
		return Context
	}
	m := make(map[string]*block)
	listId := []string{}
	p := filePath(Context.TmpPath, b.id)
	m[p] = b
	listId = append(listId, p)
	for b.previous != nil {
		b = b.previous
		p = filePath(Context.TmpPath, b.id)
		m[p] = b
		listId = append(listId, p)
	}
	Context.fileNames = m
	Context.tempList = listId
	return Context
}
func partFile(blockSize int64, length int64, start int64, end int64) *block {
	if length/(RULE*blockSize) > 0 && length/(LEVEL*LEVEL*LEVEL) == 0 {
		length = length - RULE*blockSize
		return &block{id: MD5(""), start: length, end: end, previous: partFile(blockSize, length, start, length-1)}
	}
	if length/(LEVEL*LEVEL*LEVEL) > 0 && length/(LEVEL*LEVEL*LEVEL*LEVEL) == 0 {
		length = length - RULE*blockSize
		return &block{id: MD5(""), start: length, end: end, previous: partFile(blockSize, length, start, length-1)}
	}
	if length/(LEVEL*LEVEL*LEVEL*LEVEL) > 0 {
		length = length - RULE*blockSize
		return &block{id: MD5(""), start: length, end: end, previous: partFile(blockSize, length, start, length-1)}
	}
	return &block{id: MD5(""), start: start, end: end}
}

func partFileManual(length int64, thread int64, name string) (b *block) {
	blockSize := length / thread
	surplus := length % thread
	b = nil
	var start int64
	var i int64
	if surplus == 0 {
		for i = 1; i <= thread; i++ {
			seg := new(block)
			r := name + MD5(strconv.FormatInt(i, 10))
			seg.id = r
			seg.previous = b
			seg.start = start
			seg.end = blockSize*i - 1
			start = blockSize * i
			b = seg
		}
	} else {
		for i = 1; i <= thread+1; i++ {
			seg := new(block)
			r := name + MD5(strconv.FormatInt(i, 10))
			seg.id = r
			seg.previous = b
			seg.start = start
			if i == (thread + 1) {
				seg.end = blockSize*(i-1) + surplus - 1
			} else {
				seg.end = blockSize*i - 1
			}
			start = blockSize * i
			b = seg
		}
	}
	return b
}

// 生成32位MD5
func MD5(text string) string {
	ctx := md5.New()
	if text == "" {
		ctx.Write([]byte(GetEndName()))
		return hex.EncodeToString(ctx.Sum(nil))
	}
	ctx.Write([]byte(text))
	return hex.EncodeToString(ctx.Sum(nil))
}
func GetEndName() string {
	Count++
	return strconv.Itoa(Count)
}
func createFile(file string) (f *os.File, err error) {
	if checkFileStat(file) {
		deleteFile(file)
	}
	f, err = os.Create(file)
	if err != nil {
		log.Println(file, "文件创建失败")
	}
	return f, err
}
func createFileOnly(file string) error {
	if checkFileStat(file) {
		deleteFile(file)
	}
	f, err := os.Create(file)
	if err != nil {
		log.Println(file, "文件创建失败")
	}
	defer f.Close()
	return err
}

func deleteFile(file string) error {
	if !checkFileStat(file) {
		return nil
	}
	err := os.Remove(file)
	if err != nil {
		log.Println(file, "文件删除失败")
	}
	return err
}
func checkFileStat(file string) bool {
	var exist = true
	if _, err := os.Stat(file); os.IsNotExist(err) {
		exist = false
	}
	return exist
}
func checkBlockStat(filePath string, b *block) bool {
	m := checkFileStat(filePath)
	if m {
		if int64(len(readFile(filePath))) == (b.end - b.start + 1) {
			return true
		} else {
			deleteFile(filePath)
			return false
		}
	}
	return false
}
func appendToFile(fileName string, content string) error {
	// 以只写的模式，打开文件
	f, err := os.OpenFile(fileName, os.O_WRONLY, 0644)
	if err != nil {
		log.Println("file append failed. err: " + err.Error())
	} else {
		// 查找文件末尾的偏移量
		n, _ := f.Seek(0, os.SEEK_END)
		// 从末尾的偏移量开始写入内容
		_, err = f.WriteAt([]byte(content), n)
	}
	defer f.Close()
	return err
}
func readFile(path string) []byte {
	fi, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fi.Close()
	fd, _ := ioutil.ReadAll(fi)
	return fd
}
func filePath(root, id string) string {
	t := runtime.GOOS
	var file string
	if t == WINDOWS {
		file = filepath.Join(root, id)
	}
	if t == LINUX {
		file = path.Join(root, id)
	}
	return file
}
func getFileSize(file string) (int64, error) {
	if !checkFileStat(file) {
		return 0, nil
	}
	fi, err := os.Stat(file)
	if err != nil {
		log.Println("stat err", file, err)
		return 0, err
	}
	return fi.Size(), nil
}
