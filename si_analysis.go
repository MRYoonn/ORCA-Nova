package main

// Offline SI document evidence extraction. Nothing is fetched from a DOI and
// neither PDF nor research text is uploaded to any outside service.
import (
 "bytes"
 "encoding/base64"
 "encoding/json"
 "fmt"
 "io"
 "net/http"
 "regexp"
 "sort"
 "strings"
 "unicode"

 pdf "rsc.io/pdf"
)

type SIDocument struct {
 Name string `json:"name"`
 DOI string `json:"doi"`
 Text string `json:"text"`
 PDFBase64 string `json:"pdfBase64"`
 Input string `json:"input"`
}
type SIPage struct {
 Number int `json:"page"`
 Text string `json:"text"`
}
type SIEvidence struct {
 Field string `json:"field"`
 Value string `json:"value"`
 Page int `json:"page"`
 Excerpt string `json:"excerpt"`
}
type SIField struct {
 Field string `json:"field"`
 Values []string `json:"values"`
 Status string `json:"status"`
}
type SIDifference struct {
 Field string `json:"field"`
 SI string `json:"si"`
 INP string `json:"inp"`
 Status string `json:"status"`
}
type SIResult struct {
 Name string `json:"name"`
 DOI string `json:"doi"`
 Pages int `json:"pages"`
 Evidence []SIEvidence `json:"evidence"`
 Fields []SIField `json:"fields"`
 Differences []SIDifference `json:"differences"`
 Warnings []string `json:"warnings"`
 Confirmed bool `json:"confirmed"`
}

var siKeywords = map[string][]struct{Value, Pattern string}{
 "functional":{
  {"M06-2X",`(?i)\bm[0o]6[- ]?2x\b`},
  {"M05-2X",`(?i)\bm05[- ]?2x\b`},
  {"CAM-B3LYP",`(?i)\bCAM[- ]?B3LYP\b`},
  {"B3LYP",`(?i)\bB3LYP\b`},
  {"PBE0",`(?i)\bPBE0\b`},
  {"TPSSh",`(?i)\bTPSSh\b`},
  {"TPSS",`(?i)\bTPSS\b`},
  {"BP86",`(?i)\bBP86\b`},
  {"wB97X-D3",`(?i)\b[ωw]B97X[- ]?D3\b`},
 },
 "basis":{
  {"def2-QZVPPD",`(?i)\bdef2[- ]?QZVPPD\b`},
  {"def2-QZVPP",`(?i)\bdef2[- ]?QZVPP\b`},
  {"def2-TZVPP",`(?i)\bdef2[- ]?TZVPP\b`},
  {"def2-TZVP",`(?i)\bdef2[- ]?TZVP\b`},
  {"def2-SVP",`(?i)\bdef2[- ]?SVP\b`},
  {"6-311+G(d,p)",`(?i)6[- ]?311\+G\s*\(\s*d\s*,\s*p\s*\)`},
  {"6-311G(d)",`(?i)6[- ]?311G\s*\(\s*d\s*\)`},
  {"6-31+G(d,p)",`(?i)6[- ]?31\+\s*G\s*\(\s*d\s*,\s*p\s*\)`},
 },
 "dispersion":{
  {"D3BJ",`(?i)\bD3\s*\(?\s*BJ\s*\)?\b|\bD3BJ\b|\bD3\s*[- ]?Becke[- ]?Johnson\b`},
  {"D3ZERO",`(?i)\bD3ZERO\b|\bD3\b.{0,80}\bzero[- ]?damping\b|\bD3\b.{0,60}\bzero damping\b`},
  {"D4",`(?i)\bD4\b|\bGrimme.s D4\b`},
  {"D3 (damping unspecified)",`(?i)\bD3\b|\bGrimme.s D3\b`},
 },
 "solvation":{
  {"CPCM",`(?i)\bCPCM\b|\bconductor[- ]like polariz(?:able|ation) continuum model\b`},
  {"SMD",`(?i)\bSMD\b|\bsolvation model based on density\b`},
  {"IEFPCM",`(?i)\bIEFPCM\b`},
  {"PCM",`(?i)\bPCM\b`},
 },
 "solvent":{
  {"acetonitrile",`(?i)\bacetonitrile\b|\bMeCN\b|\bCH3CN\b`},
  {"toluene",`(?i)\btoluene\b`},
  {"water",`(?i)\bwater\b`},
  {"methanol",`(?i)\bmethanol\b`},
  {"1,2,4-trichlorobenzene",`(?i)1\s*,\s*2\s*,\s*4[- ]trichlorobenzene\b|\bTCB\b`},
  {"diethyl ether",`(?i)\bEt2O\b|\bdiethyl ether\b`},
 },
 "software":{
  {"ORCA",`(?i)\bORCA\s*(?:[456]\.[0-9.]+)?\b`},
  {"Gaussian",`(?i)\bGaussian\s*(?:09|16)\b`},
 },
 "task":{
  {"Opt",`(?i)\bgeometry optimiz(?:ations?|ed|ation)\b|\boptimized geometr(?:y|ies)\b|\bOpt\b`},
  {"Freq",`(?i)\bfrequenc(?:y|ies)\b|\bvibrational\b|\bFreq\b`},
  {"SP",`(?i)\bsingle[- ]point\b|\bSP\b`},
  {"TDDFT",`(?i)\bTD[- ]?DFT\b|\bvertical excitations?\b`},
 },
}
var siOrder=[]string{"software","functional","basis","dispersion","solvation","solvent","task"}
var siSectionRx=regexp.MustCompile(`(?i)computational\s+(?:method(?:ology)?|details?|analys(?:is|es)|study)|DFT\s+calculations?|theoretical\s+calculations?|quantum[- ]chemical\s+calculations?`)

// The legacy PDF parser handles digitally born PDFs, not image scans/OCR.
// The text is returned page-by-page to preserve evidence and prevent invented pages.
func extractSIPDF(raw []byte)(pages []SIPage,err error){
 defer func(){if p:=recover();p!=nil {pages=nil;err=fmt.Errorf("PDF 파서가 지원하지 않는 문서입니다: %v",p)}}()
 if len(raw)<30||len(raw)>12<<20||!bytes.HasPrefix(raw,[]byte("%PDF-")){
  return nil,fmt.Errorf("PDF 용량은 12MB 이하이며 유효한 %%PDF- 형식이어야 합니다")
 }
 reader,err:=pdf.NewReader(bytes.NewReader(raw),int64(len(raw)))
 if err!=nil{return nil,fmt.Errorf("PDF 내용 추출 실패: %w",err)}
 n:=reader.NumPage()
 if n<1||n>250 {return nil,fmt.Errorf("SI는 PDF 1~250페이지까지 지원합니다 (현재 %d)",n)}
 for i:=1;i<=n;i++{
  p:=reader.Page(i)
  if p.V.IsNull(){continue}
  parts:=p.Content().Text
  // Sort by baseline first, then x coordinate. Preserve enough spaces to
  // recognize both split PDF glyphs and paragraphs after normalisation.
  sort.SliceStable(parts,func(i,j int)bool{
   if absFloat(parts[i].Y-parts[j].Y)>2.5{return parts[i].Y>parts[j].Y}
   return parts[i].X<parts[j].X
  })
  var b strings.Builder
  y:=0.0
  for j,g:=range parts {
   if j>0 {
    if absFloat(g.Y-y)>2.5 {b.WriteByte('\n')}else{b.WriteByte(' ')}
   }
   b.WriteString(g.S)
   y=g.Y
   if b.Len()>160000 {break}
  }
  text:=b.String()
  if strings.TrimSpace(text)!="" {pages=append(pages,SIPage{Number:i,Text:text})}
 }
 if len(pages)==0{return nil,fmt.Errorf("PDF에서 복사 가능한 글자를 찾지 못했습니다. 스캔 문서라면 텍스트를 붙여넣어 주세요")}
 return pages,nil
}
func absFloat(a float64)float64 {if a<0{return -a};return a}
func contextSnippet(s string,begin,end int)string{
 start:=begin-110;if start<0{start=0};finish:=end+135;if finish>len(s){finish=len(s)}
 return strings.Join(strings.Fields(s[start:finish])," ")
}
func analyzeSI(doc SIDocument)(SIResult,error){
 result:=SIResult{Name:doc.Name,DOI:strings.TrimSpace(doc.DOI),Evidence:[]SIEvidence{},Fields:[]SIField{},Differences:[]SIDifference{},Warnings:[]string{}}
 if len(doc.Name)>180||len(doc.DOI)>160||len(doc.Input)>30000{return result,fmt.Errorf("입력 이름/DOI/INP가 너무 깁니다")}
 var pages []SIPage
 if doc.PDFBase64!=""{
  raw,err:=base64.StdEncoding.DecodeString(doc.PDFBase64)
  if err!=nil{return result,fmt.Errorf("PDF base64를 읽을 수 없습니다")}
  pages,err=extractSIPDF(raw);if err!=nil{return result,err}
 }else{
  if len(doc.Text)<30||len(doc.Text)>2<<20{return result,fmt.Errorf("SI 텍스트는 30자~2MB 범위여야 합니다")}
  pages=[]SIPage{{Number:1,Text:doc.Text}}
  result.Warnings=append(result.Warnings,"텍스트 붙여넣기 모드: 실제 PDF 페이지 번호는 확인할 수 없으므로 Page 1은 입력한 텍스트 영역을 의미합니다.")
 }
 result.Pages=len(pages)
 found:=make(map[string]map[string]bool)
 for _,key:=range siOrder {found[key]=make(map[string]bool)}
 for _,p:=range pages {
  if !siSectionRx.MatchString(p.Text){continue}
  // A mention of a keyword is only a candidate, never confirmation.
  for _,field:=range siOrder{
   // To keep references and reaction SI tables from being interpreted as
   // global methods, prefer the short 1800-character region following a
   // computational-section heading.
   for _,section:=range siSectionRx.FindAllStringIndex(p.Text,-1){
    end:=section[1]+2200;if end>len(p.Text){end=len(p.Text)}
    excerpt:=p.Text[section[0]:end]
    if strings.Index(strings.ToLower(excerpt),"references")>=0{
      excerpt=strings.SplitN(excerpt,"References",2)[0]
    }
    for _,option:=range siKeywords[field]{
     re:=regexp.MustCompile(option.Pattern)
     m:=re.FindStringIndex(excerpt)
     if m==nil{continue}
     if field=="dispersion"&&option.Value=="D3 (damping unspecified)" {
      // A generic D3 mention should not introduce an artificial conflict
      // when D3BJ/D3ZERO is explicitly present in the same paragraph.
      if regexp.MustCompile(`(?i)\bD3BJ\b|\bD3\s*\(\s*BJ\s*\)|\bzero[- ]?damping\b`).MatchString(excerpt){continue}
     }
     if field=="solvation"&&option.Value=="PCM"{
      if regexp.MustCompile(`(?i)\b(?:CPCM|IEFPCM|SMD)\b`).MatchString(excerpt){continue}
     }
     // Reject shorter names nested in longer recognized method/basis names.
     if field=="functional"&&option.Value=="B3LYP"&&regexp.MustCompile(`(?i)CAM[- ]?B3LYP`).MatchString(excerpt){continue}
     if field=="basis"&&option.Value=="def2-TZVP"&&regexp.MustCompile(`(?i)def2[- ]?TZVPP\b`).MatchString(excerpt){continue}
     if found[field][option.Value]{continue}
     found[field][option.Value]=true
     result.Evidence=append(result.Evidence,SIEvidence{Field:field,Value:option.Value,Page:p.Number,Excerpt:contextSnippet(excerpt,m[0],m[1])})
    }
   }
  }
 }
 for _,field:=range siOrder{
  values:=[]string{}
  for v:=range found[field]{values=append(values,v)}
  sort.Strings(values)
  status:="not-found"
  if len(values)==1{status="candidate"}else if len(values)>1{status="ambiguous"}
  if field=="software"&&len(values)==1&&values[0]!="ORCA"{status="requires-translation"}
  if field=="basis"&&len(values)==1&&strings.HasPrefix(values[0],"6-"){status="unsupported-direct"}
  result.Fields=append(result.Fields,SIField{Field:field,Values:values,Status:status})
 }
 if len(result.Evidence)==0 {result.Warnings=append(result.Warnings,"Computational/DFT Method 구간에서 계산 조건을 추출하지 못했습니다. PDF 텍스트·페이지를 확인해 주세요.")}
 result.Warnings=append(result.Warnings,"자동 감지는 검증된 Method가 아닙니다. 각 근거 페이지, Opt/SP 수준, 용매·전하·Multiplicity를 직접 확인한 후 INP를 적용하세요.")
 if doc.Input!=""{result.Differences=compareSIInput(result.Fields,doc.Input)}
 return result,nil
}
func siInputTokens(inp string)map[string]string{
 t:=map[string]string{}
 for _,line:=range strings.Split(inp,"\n"){
  s:=strings.TrimSpace(line)
  if !strings.HasPrefix(s,"!"){continue}
  toks:=strings.Fields(strings.TrimPrefix(s,"!"))
  for _,tok:=range toks{
   for _,field:=range []string{"functional","basis","dispersion","solvation","task"}{
    for _,opt:=range siKeywords[field]{
     if field=="task"&&opt.Value=="Opt"&&!strings.EqualFold(tok,"Opt"){continue}
     if field=="task"&&opt.Value=="Freq"&&!strings.EqualFold(tok,"Freq"){continue}
     if field=="task"&&opt.Value=="SP"{continue}
     if regexp.MustCompile(opt.Pattern).MatchString(tok){t[field]=opt.Value;break}
    }
   }
   if m:=regexp.MustCompile(`(?i)^(?:CPCM|SMD)\(([^()]+)\)$`).FindStringSubmatch(tok);len(m)>1{
    t["solvent"]=strings.ToLower(m[1])
   }
  }
  break
 }
 if t["task"]==""{t["task"]="SP"}
 return t
}
func compareSIInput(fields []SIField,inp string)[]SIDifference{
 got:=siInputTokens(inp)
 diff:=[]SIDifference{}
 for _,field:=range fields{
  if field.Field=="software"{continue}
  target:="Not identified"
  if len(field.Values)>0{target=strings.Join(field.Values," / ")}
  have:=got[field.Field];if have==""{have="Not specified"}
  status:="unknown"
  if len(field.Values)>1{status="ambiguous"}else if len(field.Values)==1{
   if field.Status=="unsupported-direct"{status="unsupported"}
   else if strings.EqualFold(field.Values[0],have){status="match"}else{status="review"}
  }
  diff=append(diff,SIDifference{Field:field.Field,SI:target,INP:have,Status:status})
 }
 return diff
}
func (a *App) siRoute(w http.ResponseWriter,r *http.Request){
 if r.Method!=http.MethodPost {fail(w,http.StatusMethodNotAllowed,"POST 요청만 허용합니다");return}
 if r.URL.Path=="/api/si/composite"{a.siCompositeRoute(w,r);return}
 if r.URL.Path!="/api/si/analyze"{http.NotFound(w,r);return}
 var doc SIDocument
 if err:=json.NewDecoder(io.LimitReader(r.Body,20<<20)).Decode(&doc);err!=nil{fail(w,400,"SI 파일 요청을 읽지 못했습니다");return}
 if len(doc.PDFBase64)>17<<20{fail(w,413,"SI PDF는 12MB 이하여야 합니다");return}
 result,err:=analyzeSI(doc)
 if err!=nil{fail(w,422,err.Error());return}
 respond(w,200,result)
}

// A standalone helper returns printable evidence without depending on UI.
func siEvidenceText(r SIResult)string{
 var b strings.Builder
 fmt.Fprintf(&b,"Source: %s\nDOI (user-provided): %s\nExtracted pages: %d\nConfirmed: false\n",r.Name,r.DOI,r.Pages)
 for _,v:=range r.Fields {fmt.Fprintf(&b,"%s: %s [%s]\n",v.Field,strings.Join(v.Values,", "),v.Status)}
 for _,v:=range r.Evidence{fmt.Fprintf(&b,"PDF page %d | %s=%s | %s\n",v.Page,v.Field,v.Value,v.Excerpt)}
 for _,v:=range r.Differences{fmt.Fprintf(&b,"INP %s: %s vs %s [%s]\n",v.Field,v.SI,v.INP,v.Status)}
 for _,w:=range r.Warnings {fmt.Fprintf(&b,"Warning: %s\n",w)}
 return b.String()
}

// Keep static type use here explicit for future compile tests.
var _ io.Reader
var _ = unicode.IsLetter
