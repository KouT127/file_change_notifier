package main

import (
	"golang.org/x/sys/unix"
	"log"
	"time"
)

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
	// O_RDONLY: リードオンリー
	// perm: パーミッション 700
	fd, err := unix.Open(path, unix.O_RDONLY, 0700)
	if fd == -1 {
		return nil, err
	}
	return &fd, nil
}

func main() {
	path := "./test"
	watcher := map[int]string{}

	kq, err := kqueue()
	if err != nil {
		log.Fatal("error")
		return
	}
	fd, err := open(path)
	if err != nil {
		log.Fatal("err")
		return
	}
	watcher[*fd] = path

	// Keventを作成
	// ref: https://www.freebsd.org/cgi/man.cgi?kqueue
	// EV_ADD: イベントをkqueueに追加する
	// EV_CLEAR: イベントを取得した後、再びセットする
	// EV_ENABLE: keventをトリガーすることを許可する
	const evFlags = unix.EV_ADD | unix.EV_CLEAR | unix.EV_ENABLE

	// NOTE_WRITE: 書き込み
	// NOTE_RENAME: リネーム
	// NOTE_DELETE: 削除
	const noteFlags = unix.NOTE_WRITE | unix.NOTE_RENAME | unix.NOTE_DELETE
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
		time.Sleep(500 * time.Millisecond)
		if cnt == 0 {
			continue
		}

		event := events[0]
		path := watcher[int(ev.Ident)]
		if event.Fflags&unix.NOTE_DELETE == unix.NOTE_DELETE {
			log.Print( "delete: " + path)
		}
		if event.Fflags&unix.NOTE_WRITE == unix.NOTE_WRITE {
			log.Print("write: " + path)
		}
		if event.Fflags&unix.NOTE_RENAME == unix.NOTE_RENAME {
			log.Print("rename: " + path)
		}
	}
}
