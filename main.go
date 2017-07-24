package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/alexflint/go-arg"
)

var (
	current_torrents SeriesTorrent
)

func main() {
	var (
		err  error
		args struct {
			RES     string   `arg:"help:Resolution preference [480/720/1080]"`
			RELEASE []string `arg:"-r,help:Release group preference order."`
			Series  []string `arg:"required,positional,help:TV series to download"`
			NEW     bool     `arg:"-n,help:Only modify new torrents"`
			PATH    string   `arg:"-P,help:Path to torrent files"`
		}
	)
	arg.MustParse(&args)
	scanner := bufio.NewScanner(os.Stdin)
	for err == nil {
		if !scanner.Scan() {
			panic("fail")
		}
		exec.Command("wget", scanner.Text(), "-o", args.PATH+"/")
		process(args.PATH + "/" + path.Base(scanner.Text()))
	}
}

func process(torrentFile string) *MediaTorrent {
	var (
		mt *MetaTorrent  = new(MetaTorrent)
		vt *MediaTorrent = new(MediaTorrent)
	)
	f, _ := os.OpenFile(torrentFile, os.O_RDONLY, 755)
	mt.Load(f)
	fmt.Printf("%+v\n", mt)
	vt.Torrent = NewTorrent(*mt)
	vt.Process()
	fmt.Printf("%+v\n", *vt)
	return vt
}
