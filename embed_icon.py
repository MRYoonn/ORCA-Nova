"""Embed multi-size .ico in Windows PE as RT_ICON + RT_GROUP_ICON resources.
Works on a simple Go-built PE32+ executable with available section-table space.
"""
from pathlib import Path
import struct
import sys

src=Path(sys.argv[1]); ico=Path(sys.argv[2]); dst=Path(sys.argv[3]); exe=bytearray(src.read_bytes()); ib=ico.read_bytes()
U16=lambda buf,off:struct.unpack_from('<H',buf,off)[0]
U32=lambda buf,off:struct.unpack_from('<I',buf,off)[0]
W16=lambda buf,off,v:struct.pack_into('<H',buf,off,v)
W32=lambda buf,off,v:struct.pack_into('<I',buf,off,v)
align=lambda n,a:(n+a-1)//a*a
if U16(ib,0)!=0 or U16(ib,2)!=1:raise ValueError('Not ICO')
count=U16(ib,4); entries=[]
for i in range(count):
 o=6+i*16;w,h,c,r,planes,bpp,size,offset=struct.unpack_from('<BBBBHHII',ib,o)
 data=ib[offset:offset+size];assert len(data)==size
 entries.append((w,h,c,r,planes,bpp,data))

pe=U32(exe,0x3c);assert exe[pe:pe+4]==b'PE\0\0';coff=pe+4;numsec=U16(exe,coff+2)
opt=coff+20;sizeopt=U16(exe,coff+16);assert U16(exe,opt)==0x20b
sec_start=opt+sizeopt;file_align=U32(exe,opt+36);section_align=U32(exe,opt+32)
secs=[]
for i in range(numsec):
 off=sec_start+i*40
 name=exe[off:off+8].rstrip(b'\0').decode(errors='replace')
 virtualsz,virtaddr,rawsz,rawptr=struct.unpack_from('<IIII',exe,off+8)
 secs.append((name,virtualsz,virtaddr,rawsz,rawptr))
print('sections:',secs)
first_raw=min(x[4] for x in secs if x[4]); newhdr=sec_start+numsec*40
if newhdr+40>first_raw:raise ValueError('No section header space')
last_vend=max(va+align(max(vsz,rawsz),section_align) for _,vsz,va,rawsz,ptr in secs)
new_va=align(last_vend,section_align);raw_ptr=align(len(exe),file_align)
# resources directory entries relative to section RVA
buf=bytearray();
def reserve(n):
 pos=align(len(buf),4);buf.extend(bytes(pos+n-len(buf)));return pos

def u16(pos,x):struct.pack_into('<H',buf,pos,x)
def u32(pos,x):struct.pack_into('<I',buf,pos,x)
def directory(num):
 off=reserve(16+num*8);u16(off+14,num);return off

def dirEntry(dirpos,index,id,target,sub=True):
 u32(dirpos+16+index*8,id)
 u32(dirpos+20+index*8,target|(0x80000000 if sub else 0))
root=directory(2)
icon_dir=directory(count);group_dir=directory(1)
dirEntry(root,0,3,icon_dir);dirEntry(root,1,14,group_dir)
langs=[]
for i in range(count):
 sub=directory(1);dirEntry(icon_dir,i,i+1,sub);langs.append(sub)
subgroup=directory(1);dirEntry(group_dir,0,1,subgroup)
entry_positions=[]
for i,sub in enumerate(langs):
 p=reserve(16);dirEntry(sub,0,1033,p,False);entry_positions.append(p)
group_entry=reserve(16);dirEntry(subgroup,0,1033,group_entry,False)
for i,(*other,data) in enumerate(entries):
 off=reserve(len(data));buf[off:off+len(data)]=data
 u32(entry_positions[i],new_va+off);u32(entry_positions[i]+4,len(data))
# group icon header, GRPICONDIR + GRPICONDIRENTRY
G=bytearray(struct.pack('<HHH',0,1,count))
for i,(w,h,c,r,planes,bpp,data) in enumerate(entries):
 G.extend(struct.pack('<BBBBHHIH',w,h,c,r,planes,bpp,len(data),i+1))
off=reserve(len(G));buf[off:off+len(G)]=G
u32(group_entry,new_va+off);u32(group_entry+4,len(G))
raw_size=align(len(buf),file_align)
exe.extend(bytes(raw_ptr-len(exe)));exe.extend(buf);exe.extend(bytes(raw_size-len(buf)))
# Add section descriptor and PE metadata
hdr=bytearray(40);hdr[0:8]=b'.rsrc\0\0\0';struct.pack_into('<IIII',hdr,8,len(buf),new_va,raw_size,raw_ptr)
struct.pack_into('<I',hdr,36,0x40000040)
exe[newhdr:newhdr+40]=hdr
W16(exe,coff+2,numsec+1)
W32(exe,opt+56,align(new_va+len(buf),section_align))  # SizeOfImage
# image resource dir PE32+ DataDirectory[2]: OptionalHeader+112+16
W32(exe,opt+112+2*8,new_va)
W32(exe,opt+112+2*8+4,len(buf))
dst.write_bytes(exe)
print('wrote',dst,'resources',count,'icons','size',len(exe),'resource VA',hex(new_va))
