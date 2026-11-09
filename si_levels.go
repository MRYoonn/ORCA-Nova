package main

import (
 "regexp"
 "sort"
 "strings"
)

type SILevel struct{
 Kind string `json:"kind"`
 Page int `json:"page"`
 Values map[string][]string `json:"values"`
 Excerpt string `json:"excerpt"`
 Warnings []string `json:"warnings"`
}
var siStageMarkers=[]struct{Kind string;Rx *regexp.Regexp}{
 {"Opt+Freq",regexp.MustCompile(`(?i)\b(?:geometr(?:y|ies)\s+(?:were\s+|was\s+)?optimized|geometry optimiz(?:ation|ations)|structures?\s+(?:were\s+|was\s+)?optimized|optimized\s+at|structure optimization)\b`)},
 {"SP",regexp.MustCompile(`(?i)\b(?:single[- ]?point(?:\s+energy|\s+calculations?)?|high[- ]?level\s+SP)\b`)},
}
func extractSILevels(pages []SIPage)[]SILevel{
 levels:=[]SILevel{}
 for _,page:=range pages{
  if !siSectionRx.MatchString(page.Text){continue}
  for _,marker:=range siStageMarkers{
   offsets:=marker.Rx.FindAllStringIndex(page.Text,5)
   for _,p:=range offsets{
    begin:=p[0];end:=p[1]+650
    if end>len(page.Text){end=len(page.Text)}
    for _,other:=range siStageMarkers{
     if other.Kind==marker.Kind{continue}
     if z:=other.Rx.FindStringIndex(page.Text[p[1]:end]);z!=nil{
      end=p[1]+z[0]
     }
    }
    if end<=begin+15{continue}
    context:=page.Text[begin:end]
    values:=map[string][]string{}
    for _,field:=range []string{"functional","basis","dispersion","solvation","solvent"} {
     for _,entry:=range siKeywords[field]{
      re:=regexp.MustCompile(entry.Pattern)
      if !re.MatchString(context){continue}
      if field=="functional"&&entry.Value=="B3LYP"{
       withoutCAM:=regexp.MustCompile(`(?i)CAM[- ]?B3LYP`).ReplaceAllString(context,"")
       if !re.MatchString(withoutCAM){continue}
      }
      if field=="dispersion"&&entry.Value=="D3 (damping unspecified)"&&
       regexp.MustCompile(`(?i)\bD3BJ\b|\bD3\s*\(\s*BJ\s*\)|\bzero[- ]?damping\b`).MatchString(context){continue}
      values[field]=append(values[field],entry.Value)
     }
     if len(values[field])>0{sort.Strings(values[field])}
    }
    if len(values)==0{continue}
    warnings:=[]string{"Method 후보는 문단 위치 기반으로 추출되며, 계산 단계와 모든 보정 조건을 자동 검증하지 않습니다."}
    for k,v:=range values{if len(v)>1{warnings=append(warnings,k+": 복수 후보이므로 단계별 수동 확인 필요")}}
    levels=append(levels,SILevel{Kind:marker.Kind,Page:page.Number,Values:values,Excerpt:strings.Join(strings.Fields(context)," "),Warnings:warnings})
   }
  }
 }
 return levels
}
