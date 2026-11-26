package main

import (
	"strings"
	"testing"
)

func hatTestRequest() RunRequest {
	a:="3\nreactant\nC 0 0 0\nC 1.5 0 0\nH 0 1 0\n"
	b:="3\nproduct\nC 0 0 0\nC 1.5 0 0\nH 1.5 1 0\n"
	base:="! UKS M062X def2-TZVP def2/J RIJCOSX TightSCF CPCM(dmf) "
	freq:="\n%freq\n Temp 298.15\nend\n"
	return RunRequest{Files:[]JobFile{
		{Name:"reactant.xyz",Content:a},{Name:"product.xyz",Content:b},
		{Name:"r.inp",Content:base+"Opt Freq"+freq+"* xyzfile 0 2 reactant.xyz\n"},
		{Name:"p.inp",Content:base+"Opt Freq"+freq+"* xyzfile 0 2 product.xyz\n"},
		{Name:"ts.inp",Content:base+"Freq NEB-TS"+freq+"%neb\n NEB_END_XYZFILE \"p.xyz\"\nend\n* xyzfile 0 2 r.xyz\n"},
		{Name:"irc.inp",Content:base+"IRC\n* xyzfile 0 2 ts.xyz\n"},
	},Stages:[]Stage{
		{Name:"R",Input:"r.inp",Source:"reactant.xyz",Opt:true,HAT:"reactant"},
		{Name:"P",Input:"p.inp",Source:"product.xyz",Opt:true,HAT:"product"},
		{Name:"TS",Input:"ts.inp",Source:"r.xyz",EndSource:"p.xyz",Opt:true,HAT:"ts"},
		{Name:"IRC",Input:"irc.inp",Source:"ts.xyz",HAT:"irc"},
	}}
}

func TestHATPreflight(t *testing.T) {
	req:=hatTestRequest()
	if r:=preflight(req); !r.OK {t.Fatalf("valid workflow rejected: %v",r.Errors)}
	for _, tc:=range []struct{name string; edit func(*RunRequest)}{
		{"swapped element order",func(r *RunRequest){r.Files[1].Content="3\np\nH 1.5 1 0\nC 0 0 0\nC 1.5 0 0\n"}},
		{"wrong NEB endpoint",func(r *RunRequest){r.Files[4].Content=strings.ReplaceAll(r.Files[4].Content,"p.xyz","r.xyz")}},
		{"different solvent",func(r *RunRequest){r.Files[4].Content=strings.ReplaceAll(r.Files[4].Content,"CPCM(dmf)","CPCM(acetonitrile)")}},
		{"different temperature",func(r *RunRequest){r.Files[4].Content=strings.ReplaceAll(r.Files[4].Content,"298.15","300")}},
		{"different multiplicity",func(r *RunRequest){r.Files[4].Content=strings.ReplaceAll(r.Files[4].Content,"0 2","0 4")}},
		{"no TS frequencies",func(r *RunRequest){r.Files[4].Content=strings.ReplaceAll(r.Files[4].Content,"Freq NEB-TS","NEB-TS")}},
		{"no IRC",func(r *RunRequest){r.Stages=r.Stages[:3]}},
	} {t.Run(tc.name,func(t *testing.T){r:=hatTestRequest();tc.edit(&r);if p:=preflight(r);p.OK{t.Fatal("invalid HAT workflow accepted")}})}
}

func TestHATReportDoesNotInventMissingEnergy(t *testing.T) {
	r:=hatWorkflowReport(t.TempDir(),hatTestRequest(),nil)
	if !strings.Contains(r,"Unavailable")||strings.Contains(r,"Forward ΔG‡ ="){t.Fatalf("unexpected report: %s",r)}
}
