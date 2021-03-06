package main

import (
	"golang.org/x/sys/unix"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ref: https://github.com/fsnotify/fsnotify/blob/master/kqueue.go
// ref: https://www.freebsd.org/cgi/man.cgi?kqueue
// EV_ADD: イベントをkqueueに追加する
// EV_CLEAR: イベントを取得した後、再びセットする
// EV_ENABLE: keventをトリガーすることを許可する
const evFlags = unix.EV_ADD | unix.EV_CLEAR | unix.EV_ENABLE

// NOTE_WRITE: 書き込み
// NOTE_RENAME: リネーム
// NOTE_DELETE: 削除
const noteFlags = unix.NOTE_WRITE | unix.NOTE_ATTRIB | unix.NOTE_RENAME

func kqueue() (int, error) {
	// kqueue（カーネルイベントキュー）を作成
	kq, err := unix.Kqueue()
	if err != nil {
		return kq, err
	}
	return kq, err
}

func open(path string) (*int, error) {
	// FileまたはDirの識別子を取得
	// path: パス
	// O_RDONLY: リードオンリーで開く
	// perm: パーミッション 700
	fd, err := unix.Open(path, unix.O_RDONLY, 0700)
	if fd == -1 {
		return nil, err
	}
	return &fd, nil
}

var (
	watcher map[int]string
	mu      sync.Mutex
)

// TODO: 一度しかEventを取得できない。
func main() {
	path := "./test"
	files, err := ioutil.ReadDir(path)
	done := make(chan bool)

	watcher = map[int]string{}

	kq, err := kqueue()
	if err != nil {
		log.Fatal("error")
		return
	}
	for _, file := range files {
		go read(kq, path, file)
	}
	<-done
}

func read(kq int, path string, info os.FileInfo) {
	fp := filepath.Join(path, info.Name())
	fd, err := open(fp)
	if err != nil {
		log.Fatal("err")
		return
	}
	mu.Lock()
	watcher[*fd] = fp
	mu.Unlock()

	// Keventを作成
	ev := unix.Kevent_t{
		// openで取得した、fileやdirの識別子を指定する
		Ident: uint64(*fd),
		// EVFILT_VNODEはファイルのイベントを識別子として受け取る
		Filter: unix.EVFILT_VNODE,
		Flags:  evFlags,
		Fflags: noteFlags,
		Data:   0,
		Udata:  nil,
	}

	// タイムアウトを設定
	timeout := unix.Timespec{
		Sec:  0,
		Nsec: 0,
	}

	for {
		events := make([]unix.Kevent_t, 10)
		cnt, err := unix.Kevent(kq, []unix.Kevent_t{ev}, events, &timeout)
		if err != nil {
			log.Println("error")
		}
		time.Sleep(100 * time.Millisecond)
		if cnt == 0 {
			continue
		}

		event := events[0]
		mu.Lock()
		path := watcher[int(ev.Ident)]
		mu.Unlock()
		if event.Fflags&unix.NOTE_DELETE == unix.NOTE_DELETE {
			log.Print("delete: " + path)
		}
		if event.Fflags&unix.NOTE_WRITE == unix.NOTE_WRITE {
			log.Print("write: " + path)
		}
		if event.Fflags&unix.NOTE_RENAME == unix.NOTE_RENAME {
			log.Print("rename: " + path)
		}
	}
}
