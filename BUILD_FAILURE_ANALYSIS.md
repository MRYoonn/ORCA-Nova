# Build failure analysis — 2026-11-08

- Workflow run: https://github.com/MRYoonn/ORCA-Nova/actions/runs/42474368596
- Job: Windows Portable Build (`128712263438`)
- Result: Go test PASS; Go Windows EXE build PASS; native WebView2 host build FAIL.
- Actual error: `native_host/Program.cs(3,18): error CS0234: The type or namespace name 'Http' does not exist in the namespace 'System.Net'`.
- Root cause: the .NET Framework 4.8 C# project did not explicitly reference `System.Net.Http`.
- Fix: add `<Reference Include="System.Net.Http" />` to the WinForms host `.csproj`.
- Additional preventive fix: package only the native host EXE, DLLs and `.config` files, preserving runtime subdirectories. This matches the update installer's accepted-file whitelist and excludes `.pdb` / `.xml` debugging artifacts.
- Re-test: Windows Actions rerun required; actual ORCA calculation and taskbar pinning still need Windows on-device checks.
