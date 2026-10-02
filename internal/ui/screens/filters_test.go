package screens

import (
 "testing"
 "time"
 _ "time/tzdata"
)

func TestSearchDatesInclusiveLocalDays(t *testing.T) {
 loc,err:=time.LoadLocation("America/New_York"); if err!=nil {t.Fatal(err)}
 from,to,err:=searchDates("2026-03-08","2026-03-08",loc)
 if err!=nil {t.Fatal(err)}
 if from.Format(time.RFC3339)!="2026-03-08T00:00:00-05:00" || to.Format(time.RFC3339)!="2026-03-09T00:00:00-04:00" {t.Fatalf("bounds %v %v",from,to)}
 if to.Sub(*from)!=23*time.Hour {t.Fatal("DST day must not be forced to 24 hours")}
}

func TestSearchRejectsInvalidAndReversedDates(t *testing.T) {
 for _, pair:= range [][2]string{{"2026-02-30",""},{"","tomorrow"},{"2026-10-03","2026-10-02"}} {
  if _,_,err:=searchDates(pair[0],pair[1],time.UTC); err==nil {t.Fatalf("accepted %v",pair)}
 }
}
