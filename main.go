package main

import (
	"context"
	"fmt"
	"go_todo_app/config"
	"log"
	"net"
	"net/http"
	"os"

	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Printf("failed to terminate server: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.New()
	if err != nil {
		return err
	}
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		log.Fatalf("failed to listen port %d: %v", cfg.Port, err)
	}
	url := fmt.Sprintf("http://%s", l.Addr().String())
	log.Printf("start with: %v", url)

	s := &http.Server{
		// hi引数で受け取った net.Listener を利用するので
		// Addr フィールドは指定しない
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
		}),
	}
	eg, ctx := errgroup.WithContext(ctx)
	// 別ゴルーチンでHTTPサーバを起動する
	eg.Go(func() error {
		// ListenAndServe メソッドではなく Serve メソッドに変更する
		if err := s.Serve(l); err != nil &&
			// http.ErrServerClosed は
			// http.Server.Shutdown が正常に終了したことを示すので異常じゃない
			err != http.ErrServerClosed {
			log.Printf("failed to close: %+v", err)
			return err
		}
		return nil
	})

	//チャネルからの通知（終了通知）を待機する
	<-ctx.Done()
	if err := s.Shutdown(context.Background()); err != nil {
		log.Printf("failed to shutdown: %+v", err)
	}
	// Goメソッドで起動した別ゴルーチンの終了を待つ
	return eg.Wait()
}
