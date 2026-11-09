package main

import (
 "bytes"
 "encoding/base64"
 "encoding/json"
 "fmt"
 "math"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
)

// Short, manually checked computational-method parameters from THREE
// independent public SI documents; not a claim of comprehensive PDF
// reproduction. Source + printed SI page preserved for audit.
var publishedSICases=[]struct{
 label,url,page,excerpt string
 want map[string][]string
}{
 {
  "Jiang et al., trivalent phosphazenium SI",
  "https://files.sciengine.com/gridfs/1828685594578911232/edit_SI.pdf",
  "S25",
  "Computation details. Geometries optimized with m062x functional and 6-31+G(d,p) basis set using Gaussian. SMD implicit solvent model was used. Vibrational frequency analyses were carried out. Thermodynamic corrections at 298.15K with Shermo. Single point calculations with M06-2X and 6-311+G(d,p) basis set were performed in Gaussian 16.",
  map[string][]string{"functional":{"M06-2X"},"basis":{"6-31+G(d,p)","6-311+G(d,p)"},"solvation":{"SMD"},"software":{"Gaussian"},"task":{"Opt","Freq","SP"}},
 },
 {
  "Hollister et al., Chemical Science SI",
  "https://www.rsc.org/suppdata/d4/sc/d4sc03835e/d4sc03835e1.pdf",
  "70",
  "Theoretical Calculations. All calculations were performed using Orca 5.0.4. Geometry optimizations employed B3LYP, Grimme's D3(BJ) dispersion, def2-TZVP basis and CPCM with diethyl ether. Frequency calculations were performed. Vertical excitations were calculated by TD-DFT at CAM-B3LYP-D3(BJ)/def2-TZVP.",
  map[string][]string{"functional":{"B3LYP","CAM-B3LYP"},"basis":{"def2-TZVP"},"dispersion":{"D3BJ"},"solvation":{"CPCM"},"software":{"ORCA"},"task":{"Opt","Freq","TDDFT"}},
 },
 {
  "Cui et al., Angew supporting information",
  "https://pure.mpg.de/pubman/item/item_3679013_2/component/file_3685440/467_anie70244-sup-0001-suppmat.pdf",
  "S31",
  "Computational Details. All electronic structure calculations were performed using the ORCA 6.0 program package. Geometry optimizations were conducted at the B3LYP-D4/def2-tzvp CPCM(toluene) level with TightOpt TightSCF DefGrid3 Freq. Minima were confirmed with vibrational frequency analyses.",
  map[string][]string{"functional":{"B3LYP"},"basis":{"def2-TZVP"},"dispersion":{"D4"},"solvation":{"CPCM"},"solvent":{"toluene"},"software":{"ORCA"},"task":{"Opt","Freq"}},
 },
}

func findSIField(result SIResult,kind string)SIField{
 for _,field:=range result.Fields{if field.Field==kind{return field}}
 return SIField{}
}
func TestThreePublicSIMethodFixturesRequireHumanReview(t *testing.T){
 for _,tc:=range publishedSICases{t.Run(tc.label,func(t *testing.T){
  a,err:=analyzeSI(SIDocument{Name:tc.label,Text:tc.excerpt,Input:"! B3LYP def2-TZVP D4 TightSCF Opt Freq CPCM(toluene)\n* xyzfile 0 1 x.xyz\n"})
  if err!=nil{t.Fatal(err)}
  if a.Confirmed{t.Fatal("raw PDF text must never be claimed verified")}
  if len(a.Evidence)<4 {t.Fatalf("too few references: %#v",a.Evidence)}
  for field,values:=range tc.want {
   got:=findSIField(a,field)
   for _,want:=range values {
    found:=false
    for _,v:=range got.Values{if v==want{found=true}}
    if !found{t.Errorf("SI %s PDF page %s: missing %s=%s; got %v",tc.url,tc.page,field,want,got.Values)}
   }
   if len(got.Values)>1 &&got.Status!="ambiguous" {t.Errorf("must flag multi-method %s ambiguous: %v",field,got)}
  }
  for _,e:=range a.Evidence{if e.Page!=1||len(e.Excerpt)==0{t.Fatal("missing excerpt page",e)}}
  if strings.Contains(tc.label,"Jiang") {
   optOK,spOK:=false,false
   for _,level:=range a.Levels {
    for _,basis:=range level.Values["basis"] {
     if level.Kind=="Opt+Freq"&&basis=="6-31+G(d,p)" {optOK=true}
     if level.Kind=="SP"&&basis=="6-311+G(d,p)" {spOK=true}
    }
   }
   if !optOK||!spOK {t.Errorf("separate Opt/SP basis evidence missing: %#v",a.Levels)}
  }
 })}
}

func TestSIInputComparisonAvoidsGuessing(t *testing.T){
 src:="Computational Details. Geometry optimization B3LYP def2-TZVP D4 CPCM(toluene) and vibrational frequency calculations with ORCA 6.0."
 a,err:=analyzeSI(SIDocument{Text:src,Input:"! B3LYP def2-TZVP D4 TightSCF Opt Freq CPCM(toluene)"})
 if err!=nil{t.Fatal(err)}
 statuses:=map[string]string{}
 for _,d:=range a.Differences{statuses[d.Field]=d.Status}
 for _,field:=range []string{"functional","basis","dispersion","solvation","solvent"}{
  if statuses[field]!="match"{t.Errorf("%s: %s",field,statuses[field])}
 }
 mismatch,err:=analyzeSI(SIDocument{Text:src,Input:"! PBE0 def2-SVP D3BJ TightSCF CPCM(acetonitrile)"})
 if err!=nil{t.Fatal(err)}
 for _,d:=range mismatch.Differences{
  if d.Field=="functional"&&d.Status!="review"{t.Fatal("mismatch must request review",d)}
 }
 if _,err:=analyzeSI(SIDocument{Text:"short"});err==nil{t.Fatal("short text must fail")}
 if _,err:=extractSIPDF([]byte("fake PDF"));err==nil{t.Fatal("invalid PDF must fail")}
}

func mockComposite()SICompositeRequest{
 xyz:="3\nwater\nO 0.000 0.000 0.119262\nH 0.000 0.763239 -0.477047\nH 0.000 -0.763239 -0.477047\n"
 high:="SCF CONVERGED AFTER 8 CYCLES\nFINAL SINGLE POINT ENERGY -151.7770000000\nORCA TERMINATED NORMALLY\n"
 return SICompositeRequest{
  LowOUT:sampleORCA, LowINP:"! PBE0 def2-SVP D4 Opt Freq CPCM(acetonitrile)\n* xyzfile 0 3 water.xyz\n",
  HighOUT:high, HighINP:"! PBE0 def2-TZVPP D4 TightSCF CPCM(acetonitrile)\n* xyzfile 0 3 water.xyz\n",
  LowXYZ:xyz,HighXYZ:xyz,StationaryPoint:"ts",
 }
}
func TestCompositeGibbsValidations(t *testing.T){
 req:=mockComposite()
 r,err:=calculateSIComposite(req)
 if err!=nil{t.Fatal(err)}
 if !r.VerifiedInputs||r.Atoms!=3||r.Multiplicity!=3{t.Fatal(r)}
 expected:=-151.777+(-151.54618270+151.55083691)
 if math.Abs(r.CompositeGibbsEh-expected)>1e-8 {t.Fatalf("G comp=%.10f expected %.10f",r.CompositeGibbsEh,expected)}
 // A 90 degree rigid rotation preserves all interatomic distances.
 req.HighXYZ="3\nrotated\nO 0 0.119262 0\nH -0.763239 -0.477047 0\nH 0.763239 -0.477047 0\n"
 if _,err:=calculateSIComposite(req);err!=nil{t.Fatal("rigid rotation should match",err)}
 req.HighXYZ="3\nwrong geometry\nO 0 0.119262 0\nH -0.963239 -0.477047 0\nH 0.763239 -0.477047 0\n"
 if _,err:=calculateSIComposite(req);err==nil{t.Fatal("different conformer should fail")}
 req=mockComposite();req.HighINP=strings.Replace(req.HighINP,"0 3","1 2",1)
 if _,err:=calculateSIComposite(req);err==nil{t.Fatal("mixed charge/spin should fail")}
 req=mockComposite();req.HighINP=strings.Replace(req.HighINP,"CPCM(acetonitrile)","SMD(acetonitrile)",1)
 if _,err:=calculateSIComposite(req);err==nil{t.Fatal("different solvent models should fail")}
 req=mockComposite();req.StationaryPoint="minimum"
 if _,err:=calculateSIComposite(req);err==nil{t.Fatal("imaginary mode fails minimum")}
 req=mockComposite();req.HighOUT="FINAL SINGLE POINT ENERGY -151.777\n"
 if _,err:=calculateSIComposite(req);err==nil{t.Fatal("unconverged SP should fail")}
}

func TestSIAPISessionProtectionAndBadPDF(t *testing.T) {
 a:=&App{token:"si-local-token"}
 do:=func(path string,body any,auth bool)*httptest.ResponseRecorder{
  data,_:=json.Marshal(body)
  url:=path
  if auth{url+="?session=si-local-token"}
  w:=httptest.NewRecorder()
  r:=httptest.NewRequest(http.MethodPost,url,bytes.NewReader(data))
  if auth{r.Header.Set("X-ORCA-Session","si-local-token")}
  a.guard(a.route)(w,r);return w
 }
 request:=SIDocument{Name:"SI",Text:"Computational Details. B3LYP def2-TZVP D4 ORCA 6.0 frequency optimization."}
 if w:=do("/api/si/analyze",request,false);w.Code!=http.StatusForbidden{t.Fatal("session leak",w.Code)}
 if w:=do("/api/si/analyze",request,true);w.Code!=http.StatusOK{t.Fatal("SI text failed",w.Code,w.Body.String())}
 request.Text="";request.PDFBase64=base64.StdEncoding.EncodeToString([]byte("not PDF"))
 if w:=do("/api/si/analyze",request,true);w.Code!=422{t.Fatal("fake PDF accepted",w.Code,w.Body.String())}
 if w:=do("/api/si/composite",mockComposite(),true);w.Code!=200{t.Fatal("composite API failed",w.Code,w.Body.String())}
}
// A minimal original, uncompressed digitally-born single-page PDF fixture.
// No copyrighted text or third-party PDF binary is committed.
func makeSimpleSIPDF(text string)[]byte{
 var b bytes.Buffer
 b.WriteString("%PDF-1.4\n")
 offsets:=[]int{0}
 objects:=[]string{
  "<< /Type /Catalog /Pages 2 0 R >>",
  "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
  "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
  fmt.Sprintf("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /FirstChar 32 /LastChar 126 /Widths [ %s ] >>","300 "+strings.TrimSpace(strings.Repeat("600 ",94))),
 }
 stream:=fmt.Sprintf("BT /F1 12 Tf 50 720 Td (%s) Tj ET",strings.ReplaceAll(text,"(",""))
 objects=append(objects,fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream",len(stream),stream))
 for i,body:=range objects{
  offsets=append(offsets,b.Len())
  fmt.Fprintf(&b,"%d 0 obj\n%s\nendobj\n",i+1,body)
 }
 xref:=b.Len()
 fmt.Fprintf(&b,"xref\n0 %d\n0000000000 65535 f \n",len(offsets))
 for i:=1;i<len(offsets);i++{fmt.Fprintf(&b,"%010d 00000 n \n",offsets[i])}
 fmt.Fprintf(&b,"trailer << /Root 1 0 R /Size %d >>\nstartxref\n%d\n%%%%EOF\n",len(offsets),xref)
 return b.Bytes()
}
func TestDigitallyBornPDFFixturePerPageExtraction(t *testing.T){
 raw:=makeSimpleSIPDF("Computational Details B3LYP def2-TZVP D4 CPCM toluene")
 pages,err:=extractSIPDF(raw)
 if err!=nil{t.Fatal(err)}
 if len(pages)!=1||pages[0].Number!=1||!strings.Contains(pages[0].Text,"B3LYP"){t.Fatal(pages)}
 result,err:=analyzeSI(SIDocument{Name:"test.pdf",PDFBase64:base64.StdEncoding.EncodeToString(raw)})
 if err!=nil{t.Fatal(err)}
 if len(findSIField(result,"basis").Values)!=1{t.Fatal(result,pages)}
}
