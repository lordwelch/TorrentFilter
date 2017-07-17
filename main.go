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
	current_torrents [][]TorrentVideo
)

func main() {
	var args struct {
		P720     bool     `arg:"-7,help:Do not select 720p file if another exists."`
		NUKED    bool     `arg:"-N,help:Allow NUKED files."`
		DIVX     bool     `arg:"help:Prefer DivX encoding if available. Default x264"`
		PROPER   bool     `arg:"help:Do not prefer PROPER FILES."`
		INTERNAL bool     `arg:"help:Prefer INTERNAL files."`
		RELEASE  string   `arg:"-r,help:Release group preference order. Comma seperated."`
		SHOW     []string `arg:"positional,help:TV show to download"`
		NEW      bool     `arg:"-n,help:Only modify new torrents"`
		PATH     string   `arg:"-P,help:Path to torrent files"`
	}
	arg.MustParse(&args)
	scnr := bufio.NewScanner(os.Stdin)
	for err == nil {
		if !scnr.scan() {
			panic("fail")
		}
		exec.Command("wget", scnr.Text(), "-o", args.PATH+'/')
		process(args.PATH + '/' + path.Base(scnr.Text()))
	}
}

func process(torrentFile string) *TorrentVideo {
	var (
		mt *MetaTorrent  = new(MetaTorrent)
		vt *TorrentVideo = new(TorrentVideo)
	)
	f, _ := os.OpenFile(torrentFile, os.O_RDONLY, 755)
	mt.Load(f)
	fmt.Printf("%+v\n", mt)
	vt.Torrent = NewTorrent(*mt)
	vt.Process()
	fmt.Printf("%+v\n", *vt)
	return vt
}
