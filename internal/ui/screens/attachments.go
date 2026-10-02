package screens

import (
 "context"
 "image"
 "path/filepath"

 "fyne.io/fyne/v2"
 "fyne.io/fyne/v2/container"
 "fyne.io/fyne/v2/dialog"
 "fyne.io/fyne/v2/widget"
 "github.com/NguyenHien-8/NoteHub/internal/domain"
 "github.com/NguyenHien-8/NoteHub/internal/platform"
 "github.com/NguyenHien-8/NoteHub/internal/ui/components"
 "github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

type Attachments struct {
 Object fyne.CanvasObject
 env *Environment
 cards *fyne.Container
 more *widget.Button
 status *widget.Label
 items []domain.Attachment
 generation uint64
 loading bool
}

func NewAttachments(env *Environment)*Attachments {
 s:=&Attachments{env:env,cards:container.NewVBox(),status:widget.NewLabel("Attachments")}
 s.more=widget.NewButton("Load more",s.load);s.more.Hide()
 scroll:=container.NewVScroll(s.cards)
 s.Object=container.NewBorder(widget.NewLabelWithStyle("Attachments",fyne.TextAlignLeading,fyne.TextStyle{Bold:true}),container.NewVBox(s.status,s.more),nil,nil,scroll)
 return s
}

func(s *Attachments) Refresh(){s.generation++;s.loading=false;s.items=nil;s.cards.RemoveAll();s.more.Hide();s.load()}

type attachmentPage struct {items []domain.Attachment; thumbs map[int64]image.Image}

func(s *Attachments) load(){
 if s.loading{return};s.loading=true;s.more.Disable();s.status.SetText("Loading attachments…")
 generation,offset:=s.generation,len(s.items)
 work.Run(s.env.Jobs,func(ctx context.Context)(attachmentPage,error){
  items,err:=s.env.Backend.Attachments.ListAll(ctx,40,offset);if err!=nil{return attachmentPage{},err}
  page:=attachmentPage{items:items,thumbs:make(map[int64]image.Image)}
  for _,a:=range items{page.thumbs[a.ID]=s.env.Previews.Image(ctx,a)}
  return page,nil
 },func(page attachmentPage,err error){
  if generation!=s.generation{return};s.loading=false;s.more.Enable()
  if err!=nil{s.status.SetText("Could not load attachments");s.env.Error(err);return}
  s.items=append(s.items,page.items...)
  for _,a:=range page.items {
   s.cards.Add(components.NewAttachmentCard(a,page.thumbs[a.ID],func(){s.open(a,false)},func(){s.open(a,true)},func(){s.remove(a)},func(){s.env.Memo(a.MemoID)}))
  }
  s.status.SetText("");if len(s.items)==0{s.status.SetText("No attachments yet. Add files to a note to keep them here.")}
  if len(page.items)==40{s.more.Show()}else{s.more.Hide()}
 })
}

func(s *Attachments) open(a domain.Attachment,folder bool){
 work.Run(s.env.Jobs,func(ctx context.Context)(struct{},error){path,err:=s.env.Backend.Attachments.Path(ctx,a.ID);if err!=nil{return struct{}{},err};if folder{path=filepath.Dir(path)};return struct{}{},platform.OpenExternal(path)},func(_ struct{},err error){s.env.Error(err)})
}

func(s *Attachments) remove(a domain.Attachment){
 dialog.ShowConfirm("Delete attachment?","Remove "+a.Filename+" from this note?",func(ok bool){if !ok{return}
  work.Run(s.env.Jobs,func(ctx context.Context)(struct{},error){return struct{}{},s.env.Backend.Attachments.Delete(ctx,a.ID)},func(_ struct{},err error){if err!=nil{s.env.Error(err);return};s.env.Changed()})
 },s.env.Window)
}

// Attachment browser screen.
