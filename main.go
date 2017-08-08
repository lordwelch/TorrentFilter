package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alexflint/go-arg"
)

var (
	current_torrents SeriesTorrent
	unselectedDir    string
)

func main() {
	var (
		torrentName string
		torrentPath string
		args        struct {
			RES     string   `arg:"help:Resolution preference [480/720/1080]"`
			RELEASE []string `arg:"-r,help:Release group preference order."`
			Series  []string `arg:"required,positional,help:TV series to download"`
			NEW     bool     `arg:"-n,help:Only modify new torrents"`
			PATH    string   `arg:"-P,help:Path to torrent files"`
		}
	)
	arg.MustParse(&args)
	unselectedDir = filepath.Clean(args.PATH + "/unselected/")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		torrentName = filepath.Base(url)
		torrentPath = filepath.Join(unselectedDir, torrentName)
		cmd := exec.Command("wget", url, "-o", torrentPath)
		if cmd.Run() != nil {
			fmt.Println("url failed: ", url)
			continue
		}
		process(torrentPath)
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
