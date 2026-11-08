#!/usr/bin/python3
import os,sys,time,re,pathlib
inp=pathlib.Path(sys.argv[1]);text=inp.read_text();time.sleep(.7)
if pathlib.Path.cwd().name.startswith('mol2_') and 'xyzfile 0 3 ' in text and not pathlib.Path('../allow_retry').exists():
 print('SCF NOT CONVERGED');sys.exit(0)
print('SCF CONVERGED AFTER 10 CYCLES')
if re.search(r'\bOpt\b',text,re.I):
 print('THE OPTIMIZATION HAS CONVERGED')
 source=re.search(r'\*\s*xyzfile\s+\S+\s+\S+\s+(\S+)',text)[1]
 inp.with_suffix('.xyz').write_bytes(pathlib.Path(source).read_bytes())
print('FINAL SINGLE POINT ENERGY -75.1234\nMULLIKEN ATOMIC CHARGES AND SPIN POPULATIONS\n0 O : -0.4 0.125\n1 H : 0.2 -0.25\n2 H : 0.2 0.125\n\nORCA TERMINATED NORMALLY')
