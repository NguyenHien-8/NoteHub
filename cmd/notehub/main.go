package main

import (
 "context"
 "flag"
 "fmt"
 "log"

 "fyne.io/fyne/v2"
 fyneapp "fyne.io/fyne/v2/app"
 "fyne.io/fyne/v2/container"
 "fyne.io/fyne/v2/widget"
 "github.com/NguyenHien-8/NoteHub/assets"
 "github.com/NguyenHien-8/NoteHub/internal/app"
 "github.com/NguyenHien-8/NoteHub/internal/ui"
)

var version="0.2.0"

func main(){
 dataDir:=flag.String("data-dir","","Use a separate NoteHub data directory")
 printVersion:=flag.Bool("version",false,"Print version and exit")
 flag.Parse()
 if *printVersion{fmt.Println("NoteHub "+version);return}
 application:=fyneapp.NewWithID("io.notehub.desktop");application.SetIcon(assets.Logo)
 backend,err:=app.OpenBackend(context.Background(),app.Config{AppName:"NoteHub",Version:version,DataDir:*dataDir})
 if err!=nil {
  log.Printf("NoteHub startup failed: %v",err)
  window:=application.NewWindow("NoteHub · Unable to start")
  message:=widget.NewLabel("NoteHub could not open its local data.\n\n"+err.Error());message.Wrapping=fyne.TextWrapWord
  window.SetContent(container.NewBorder(nil,widget.NewButton("Close",window.Close),nil,nil,message));window.Resize(fyne.NewSize(620,260));window.ShowAndRun();return
 }
 defer func(){if err:=backend.Close();err!=nil{log.Printf("Closing NoteHub database: %v",err)}}()
 desktop:=ui.NewWindow(application,backend,version)
 desktop.ShowAndRun()
 if err:=desktop.Wait();err!=nil{log.Printf("Stopping NoteHub: %v",err)}
}
