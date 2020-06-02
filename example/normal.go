package main

import "github.com/bary321/gorc"

func main() {
	//var url = "http://down.xp696.com/17.1/win10_64/DEEP_Win10x64_cjb201701.rar"
	//var url = "https://github.com/yangyangwithgnu/goagent_out_of_box_yang/archive/master.zip"
	//var url = "http://down.ylmf123.com/17.1/win7_32/YLMF123_Win7x86_201701.rar"
	//var url = "http://down.360safe.com/se/360se8.2.1.332.exe"
	var url = "https://ipfs.io/ipfs/QmdXyqbmy2bkJA9Kyhh6z25GrTCq48LwX6c1mxPsm54wi7"
	gorc.Download(url, 5, false, "/tmp", 1, 0, "testfile")
}
