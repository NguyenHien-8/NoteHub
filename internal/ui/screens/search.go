package screens

import (
 "context"
 "fmt"
 "image"
 "strings"
 "time"

 "fyne.io/fyne/v2"
 "fyne.io/fyne/v2/container"
 "fyne.io/fyne/v2/widget"
 "github.com/NguyenHien-8/NoteHub/internal/domain"
 "github.com/NguyenHien-8/NoteHub/internal/repository"
 "github.com/NguyenHien-8/NoteHub/internal/ui/components"
 "github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

type Search struct {
 Object fyne.CanvasObject
 Query *widget.Entry
 Tag,From,To *widget.Entry
 env *Environment
 list *components.TimelineList
 status *widget.Label
 timer *time.Timer
 generation uint64
 loading bool
 query repository.SearchQuery
 items []domain.Memo
 thumbs map[int64]image.Image
}

func NewSearch(env *Environment)*Search {
 s:=&Search{env:env,list:components.NewTimelineList(),status:widget.NewLabel("Search your notes"),Tag:widget.NewEntry(),From:widget.NewEntry(),To:widget.NewEntry()}
 s.Query=components.NewSearchBar(func(string){s.schedule()})
 s.Tag.SetPlaceHolder("Tag (optional)");s.From.SetPlaceHolder("From: YYYY-MM-DD");s.To.SetPlaceHolder("To: YYYY-MM-DD")
 s.Tag.OnChanged=func(string){s.schedule()};s.From.OnChanged=s.Tag.OnChanged;s.To.OnChanged=s.Tag.OnChanged
 filters:=container.NewGridWithColumns(3,s.Tag,s.From,s.To)
 header:=components.Surface(container.NewVBox(widget.NewLabelWithStyle("Search",fyne.TextAlignLeading,fyne.TextStyle{Bold:true}),s.Query,filters,s.status))
 s.Object=container.NewBorder(header,nil,nil,nil,s.list.Object)
 return s
}

func(s *Search) schedule(){
 s.generation++;generation:=s.generation
 if s.timer!=nil{s.timer.Stop()}
 s.list.SetMore(false,nil)
 s.timer=time.AfterFunc(300*time.Millisecond,func(){s.env.Jobs.Post(func(){if generation==s.generation{s.Refresh()}})})
}

func(s *Search) Close(){if s.timer!=nil{s.timer.Stop()}}

func(s *Search) Refresh(){
 s.generation++;s.loading=false
 if s.timer!=nil{s.timer.Stop()}
 from,to,err:=searchDates(s.From.Text,s.To.Text,time.Local)
 s.items=nil;s.thumbs=make(map[int64]image.Image);s.list.SetMemos(nil,nil,s.env.Actions);s.list.SetMore(false,nil)
 if err!=nil{s.status.SetText(err.Error());return}
 s.query=repository.SearchQuery{Text:s.Query.Text,From:from,To:to,Limit:40}
 if tag:=strings.TrimPrefix(strings.TrimSpace(s.Tag.Text),"#");tag!=""{s.query.Tags=[]string{tag}}
 s.load()
}

func(s *Search) load(){
 if s.loading{return};s.loading=true;s.status.SetText("Searching…")
 generation,q:=s.generation,s.query;q.Offset=len(s.items)
 work.Run(s.env.Jobs,func(ctx context.Context)(memoPage,error){
  items,err:=s.env.Backend.Search.Search(ctx,q);if err!=nil{return memoPage{},err}
  return memoPage{items:items,thumbs:s.env.Previews.Memos(ctx,items)},nil
 },func(page memoPage,err error){
  if generation!=s.generation{return};s.loading=false
  if err!=nil{s.status.SetText("Search could not be completed");s.env.Error(err);return}
  s.items=append(s.items,page.items...);for id,img:=range page.thumbs{s.thumbs[id]=img}
  s.status.SetText(fmt.Sprintf("%d notes found",len(s.items)))
  s.list.SetMemos(s.items,s.thumbs,s.env.Actions);s.list.SetMore(len(page.items)==40,s.load)
 })
}

// Search screen.
