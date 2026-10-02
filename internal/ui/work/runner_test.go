package work

import (
 "context"
 "errors"
 "testing"
 "time"
)

func TestStoppedRunnerDiscardsQueuedUIUpdates(t *testing.T) {
 queue := make(chan func(), 4)
 r := NewWithDispatcher(func(f func()) { queue <- f })
 delivered := false
 Run(r, func(context.Context) (int,error) { return 42,nil }, func(v int,err error) { delivered=true })
 var callback func()
 select { case callback= <-queue: case <-time.After(time.Second): t.Fatal("worker did not finish") }
 r.Stop()
 r.Wait()
 callback()
 if delivered { t.Fatal("widget callback ran after window closed") }
}

func TestRunnerCancelsAndWaitsForWork(t *testing.T) {
 r := NewWithDispatcher(func(f func()) { f() })
 started := make(chan struct{})
 done := make(chan error,1)
 Run(r,func(ctx context.Context)(int,error) { close(started); <-ctx.Done(); done<-ctx.Err(); return 0,ctx.Err() },func(int,error){})
 <-started
 r.Stop()
 r.Wait()
 if err := <-done; !errors.Is(err,context.Canceled) { t.Fatal(err) }
 if r.Busy() { t.Fatal("pending operations after Wait") }
 called := false
 Run(r,func(context.Context)(int,error){ called=true; return 0,nil },func(int,error){})
 r.Wait()
 if called { t.Fatal("new work accepted after Stop") }
}
