package dialogs

import (
 "context"
 "errors"
 "fmt"

 "fyne.io/fyne/v2"
 "fyne.io/fyne/v2/container"
 "fyne.io/fyne/v2/dialog"
 "fyne.io/fyne/v2/theme"
 "fyne.io/fyne/v2/widget"
 "github.com/NguyenHien-8/NoteHub/internal/domain"
 "github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

func(m *Manager) ShowMemo(id int64){
 work.Run(m.Jobs,func(ctx context.Context)(*domain.Memo,error){return m.Backend.Memos.Get(ctx,id)},func(note *domain.Memo,err error){
  if err!=nil{m.Error(err);return}
  text:=widget.NewRichTextWithText(note.Content);text.Wrapping=fyne.TextWrapWord
  files:=container.NewVBox()
  for _,a:=range note.Attachments{files.Add(widget.NewButtonWithIcon(a.Filename,theme.FileIcon(),func(){m.OpenAttachment(a)}))}
  var d *dialog.CustomDialog
  edit:=widget.NewButtonWithIcon("Edit",theme.DocumentCreateIcon(),func(){d.Hide();m.EditMemo(*note)})
  attach:=widget.NewButtonWithIcon("Attach files",theme.MailAttachmentIcon(),func(){m.AttachToMemo(id,func(){d.Hide();m.ShowMemo(id)})})
  share:=widget.NewButtonWithIcon("Share",theme.MailForwardIcon(),func(){m.ShareMemo(*note)})
  body:=container.NewBorder(container.NewHBox(edit,attach,share),files,nil,nil,container.NewVScroll(text))
  d=dialog.NewCustom("Note · "+note.CreatedAt.Local().Format("02 Jan 2006 15:04"),"Close",body,m.Window)
  d.Resize(fyne.NewSize(720,540));d.Show()
 })
}

func(m *Manager) EditMemo(note domain.Memo){
 work.Run(m.Jobs,func(ctx context.Context)(*domain.Memo,error){return m.Backend.Memos.Get(ctx,note.ID)},func(current *domain.Memo,err error){
  if err!=nil{m.Error(err);return}
  entry:=widget.NewMultiLineEntry();entry.Wrapping=fyne.TextWrapWord;entry.SetText(current.Content)
  m.drafts[entry]=current.Content
  status:=widget.NewLabel("");status.Wrapping=fyne.TextWrapWord
  var d *dialog.CustomDialog
  var save,reload,cancel *widget.Button
  busy:=false
  cancel=widget.NewButton("Cancel",func(){if busy{return};if entry.Text==m.drafts[entry]{d.Hide();return};dialog.ShowConfirm("Discard edits?","Your unsaved changes will be lost.",func(ok bool){if ok{d.Hide()}},m.Window)})
  save=widget.NewButtonWithIcon("Save",theme.DocumentSaveIcon(),func(){
   if busy{return};busy=true;save.Disable();reload.Disable();cancel.Disable();entry.Disable()
   content,revision:=entry.Text,current.Revision
   work.Run(m.Jobs,func(ctx context.Context)(*domain.Memo,error){return m.Backend.Memos.Update(ctx,current.ID,revision,content)},func(updated *domain.Memo,err error){
    busy=false;save.Enable();reload.Enable();cancel.Enable();entry.Enable()
    if errors.Is(err,domain.ErrConflict){status.SetText("Ghi chú đã được thay đổi ở nơi khác. Vui lòng tải lại. Your edits are kept here; copy them before reloading.");return}
    if err!=nil{m.Error(err);return};d.Hide();m.Changed()
   })
  });save.Importance=widget.HighImportance
  reload=widget.NewButton("Reload latest",func(){dialog.ShowConfirm("Reload note?","Copy any edits you want to keep before replacing this editor with the latest saved note.",func(ok bool){if !ok{return};work.Run(m.Jobs,func(ctx context.Context)(*domain.Memo,error){return m.Backend.Memos.Get(ctx,current.ID)},func(latest *domain.Memo,err error){if err!=nil{m.Error(err);return};current=latest;entry.SetText(latest.Content);m.drafts[entry]=latest.Content;status.SetText(fmt.Sprintf("Loaded revision %d",latest.Revision))})},m.Window)})
  body:=container.NewBorder(nil,container.NewVBox(status,container.NewHBox(reload,cancel,save)),nil,nil,entry)
  d=dialog.NewCustomWithoutButtons("Edit note",body,m.Window)
  d.SetOnClosed(func(){delete(m.drafts,entry)})
  d.Resize(fyne.NewSize(720,540));d.Show();m.Window.Canvas().Focus(entry)
 })
}

func(m *Manager) PickTag(done func(string)){
 entry:=widget.NewEntry();entry.SetPlaceHolder("Research/FPGA")
 work.Run(m.Jobs,func(ctx context.Context)([]domain.TagCount,error){return m.Backend.Tags.List(ctx)},func(tags []domain.TagCount,err error){
  if err!=nil{m.Error(err);return}
  options:=make([]string,0,len(tags));for _,tag:=range tags{options=append(options,tag.Tag)}
  selectTag:=widget.NewSelect(options,func(tag string){entry.SetText(tag)})
  d:=dialog.NewForm("Add tag","Insert tag","Cancel",[]*widget.FormItem{widget.NewFormItem("Existing",selectTag),widget.NewFormItem("Tag",entry)},func(ok bool){if ok{done(entry.Text)}},m.Window)
  d.Resize(fyne.NewSize(420,240));d.Show()
 })
}
