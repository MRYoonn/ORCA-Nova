package main

import (
 "bytes"
 "encoding/json"
 "fmt"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestUpdateChannelsSemverAndOptIn(t *testing.T){
 for _,tc:=range []struct{candidate,current string;want bool}{
  {"v1.6.0-rc.8","1.6.0-rc.7",true},
  {"v1.6.0-rc.10","1.6.0-rc.9",true},
  {"v1.6.0-rc.9","1.6.0-rc.10",false},
  {"v1.6.0-rc.8","1.6.0",false},
  {"v1.6.0","1.6.0-rc.8",true},
  {"v1.6.0","1.5.1",true},
  {"v1.6.1-rc.1","1.6.0",true},
  {"v1.5.1","1.6.0",false},
  {"v1.6.0-preview.3","1.6.0-rc.8",false},
 }{
  if newerVersion(tc.candidate,tc.current)!=tc.want{
   t.Errorf("newerVersion(%s,%s) != %t",tc.candidate,tc.current,tc.want)
  }
 }
 makeRel:=func(tag string,pre bool) githubRelease{
  return githubRelease{Tag:tag,Prerelease:pre,Assets:[]updateAsset{{Name:releaseZipName},{Name:"SHA256SUMS.txt"}}}
 }
 candidates:=[]githubRelease{makeRel("v1.6.0-rc.8",true),makeRel("v1.6.0-rc.10",true),makeRel("v1.6.0-rc.9",true)}
 rel,ok:=selectRelease(candidates,"beta","1.6.0-rc.8")
 if !ok||rel.Tag!="v1.6.0-rc.10"{t.Fatalf("beta RC upgrade failed %#v %v",rel,ok)}
 if _,ok=selectRelease(candidates,"stable","1.5.1");ok{t.Fatal("stable must never offer prerelease")}
 candidates=append(candidates,makeRel("v1.6.0",false))
 rel,ok=selectRelease(candidates,"beta","1.6.0-rc.10")
 if !ok||rel.Tag!="v1.6.0"{t.Fatal("beta should get finalized release",rel)}
 candidates=append(candidates,makeRel("v1.6.1-rc.1",true))
 rel,ok=selectRelease(candidates,"stable","1.5.1")
 if !ok||rel.Tag!="v1.6.0"{t.Fatal("stable must choose highest stable",rel)}
 candidates=append(candidates,githubRelease{Tag:"v1.7.0",Prerelease:false})
 rel,ok=selectRelease(candidates,"stable","1.5.1")
 if !ok||rel.Tag!="v1.6.0"{t.Fatal("missing ZIP/checksum must never be offered",rel)}
}

func TestUpdaterBetaGitHubListingAndSwitchClearsCachedOffer(t *testing.T){
 u:=newUpdater()
 releases:=[]githubRelease{
  {Tag:"v1.6.0-rc.8",Prerelease:true,HTMLURL:"https://github.com/MRYoonn/ORCA-Nova/releases/tag/v1.6.0-rc.8",Assets:[]updateAsset{{Name:releaseZipName},{Name:"SHA256SUMS.txt"}}},
  {Tag:"v1.5.1",Prerelease:false,Assets:[]updateAsset{{Name:releaseZipName},{Name:"SHA256SUMS.txt"}}},
 }
 u.client.Transport=updaterTransport(func(req *http.Request)(*http.Response,error){
  if req.URL.Host!="api.github.com"{return nil,fmt.Errorf("not GitHub")}
  if req.URL.Path!="/repos/"+updateRepository+"/releases"{return nil,fmt.Errorf("not beta releases endpoint: %s",req.URL.Path)}
  if req.URL.Query().Get("per_page")!="50"{return nil,fmt.Errorf("wrong pagination")}
  data,_:=json.Marshal(releases)
  return updateResponse(http.StatusOK,string(data)),nil
 })
 if state,err:=u.setChannel("beta");err!=nil||state.Channel!="beta"{t.Fatal(state,err)}
 state:=u.check()
 if !state.Available||state.Latest!="v1.6.0-rc.8"||state.Channel!="beta"||!state.Prerelease{
  t.Fatal("beta pre-release not surfaced correctly",state)
 }
 state,err:=u.setChannel("stable")
 if err!=nil||state.Available||state.ReleaseURL!=""||state.Latest!=""||state.Prerelease{
  t.Fatal("channel switch retained stale installable release",state,err)
 }
 if _,err:=u.setChannel("nightly");err==nil{t.Fatal("invalid update channel accepted")}
 if state.Channel!="stable"{t.Fatal("invalid channel modified state")}
 // Beta request errors cannot leave the previous release installable.
 _,_ = u.setChannel("beta")
 u.client.Transport=updaterTransport(func(*http.Request)(*http.Response,error){return nil,fmt.Errorf("offline")})
 state=u.check()
 if state.Available||state.ReleaseURL!=""||state.Latest!="" {t.Fatal("offline stale release",state)}
}

func TestUpdaterChannelPersistenceAndSessionAuth(t *testing.T){
 a:=&App{cfg:Config{WorkRoot:t.TempDir(),UpdateChannel:"stable"},cfgPath:filepath.Join(t.TempDir(),"config.json"),token:"channel-token",updates:newUpdater()}
 api:=func(channel string,authenticated bool) *httptest.ResponseRecorder{
  body,_:=json.Marshal(map[string]string{"channel":channel})
  url:="/api/update/channel"
  if authenticated{url+="?session=channel-token"}
  req:=httptest.NewRequest(http.MethodPost,url,bytes.NewReader(body))
  if authenticated{req.Header.Set("X-ORCA-Session","channel-token")}
  res:=httptest.NewRecorder()
  a.guard(a.route)(res,req)
  return res
 }
 if result:=api("beta",false);result.Code!=403{t.Fatalf("untrusted channel mutation: %d",result.Code)}
 if result:=api("nightly",true);result.Code!=400{t.Fatal("invalid channel accepted",result.Code)}
 if result:=api("beta",true);result.Code!=200{t.Fatalf("beta switch rejected: %d %s",result.Code,result.Body.String())}
 data,err:=os.ReadFile(a.cfgPath)
 if err!=nil||!strings.Contains(string(data),`"updateChannel": "beta"`){t.Fatal("beta choice not persisted",err,string(data))}
 if st:=a.updates.snapshot();st.Channel!="beta"{t.Fatal("runtime and persisted channel out of sync",st)}
 if result:=api("stable",true);result.Code!=200{t.Fatal("cannot restore stable",result.Body.String())}
 if a.cfg.UpdateChannel!="stable"{t.Fatal("stable not saved")}
}
