package screens

import (
 "fmt"
 "strings"
 "time"
)

func searchDates(fromText,toText string,loc *time.Location)(from,to *time.Time,err error) {
 if loc==nil {loc=time.Local}
 if text:=strings.TrimSpace(fromText); text!="" {
  day,e:=time.ParseInLocation("2006-01-02",text,loc); if e!=nil {return nil,nil,fmt.Errorf("From date must be YYYY-MM-DD")}; from=&day
 }
 if text:=strings.TrimSpace(toText); text!="" {
  day,e:=time.ParseInLocation("2006-01-02",text,loc); if e!=nil {return nil,nil,fmt.Errorf("To date must be YYYY-MM-DD")}; day=day.AddDate(0,0,1); to=&day
 }
 if from!=nil && to!=nil && !from.Before(*to) {return nil,nil,fmt.Errorf("From date must be on or before To date")}
 return from,to,nil
}
