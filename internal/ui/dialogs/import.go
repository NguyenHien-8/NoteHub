package dialogs

import (
 "context"
 "fmt"

 "fyne.io/fyne/v2"
 "fyne.io/fyne/v2/container"
 "fyne.io/fyne/v2/dialog"
 "fyne.io/fyne/v2/widget"
 "github.com/NguyenHien-8/NoteHub/internal/platform"
 "github.com/NguyenHien-8/NoteHub/internal/service"
 "github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

func(m *Manager) Import(){
 if m.backupBusy{return};m.backupBusy=true
 work.Run(m.Jobs,platform.ChooseBackup,func(path string,err error){
  if err!=nil{m.backupBusy=false;if !platform.DialogCanceled(err){m.Error(err)};return}
  policy:=widget.NewSelect([]string{"Skip","Replace","Duplicate"},nil);policy.SetSelected("Skip")
  text:=widget.NewLabel("Choose what happens when a note already exists. Replace overwrites its saved content and attachments. Export a backup first if you need to keep the current version.");text.Wrapping=fyne.TextWrapWord
  d:=dialog.NewCustomConfirm("Import backup","Import","Cancel",container.NewVBox(text,policy),func(ok bool){
   if !ok{m.backupBusy=false;return}
   selected:=map[string]service.ImportPolicy{"Skip":service.ImportSkip,"Replace":service.ImportReplace,"Duplicate":service.ImportDuplicate}[policy.Selected]
   progress:=widget.NewProgressBarInfinite();busy:=dialog.NewCustomWithoutButtons("Importing backup",container.NewVBox(widget.NewLabel("Validating archive and restoring notes…"),progress),m.Window);busy.Show()
   work.Run(m.Jobs,func(ctx context.Context)(*service.ImportReport,error){return m.Backend.Backup.ImportFile(ctx,path,selected)},func(report *service.ImportReport,err error){
    m.backupBusy=false;progress.Stop();busy.Hide();m.Changed()
    if err!=nil{m.Error(err);return}
    summary:=fmt.Sprintf("Created: %d\nReplaced: %d\nDuplicated: %d\nSkipped: %d",report.Created,report.Replaced,report.Duplicated,report.Skipped)
    if len(report.Failures)>0{summary+="\n\nFailed notes:";for _,failure:=range report.Failures{summary+="\n"+failure.UID+": "+failure.Message}}
    label:=widget.NewLabel(summary);label.Wrapping=fyne.TextWrapWord
    result:=dialog.NewCustom("Import results","Close",container.NewVScroll(label),m.Window);result.Resize(fyne.NewSize(600,400));result.Show()
   })
  },m.Window);d.Resize(fyne.NewSize(580,280));d.Show()
 })
}

// Import backup dialog.
