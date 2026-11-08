using System;
using System.Drawing;
using System.Net.Http;
using System.Runtime.InteropServices;
using System.Threading.Tasks;
using System.Windows.Forms;
using Microsoft.Web.WebView2.WinForms;

namespace ORCANovaUI {
    internal static class Program {
        [DllImport("shell32.dll", CharSet=CharSet.Unicode)]
        private static extern int SetCurrentProcessExplicitAppUserModelID(string appID);
        [STAThread]
        private static void Main(string[] args) {
            Application.EnableVisualStyles();
            Application.SetCompatibleTextRenderingDefault(false);
            SetCurrentProcessExplicitAppUserModelID("MRYoonn.ORCANova");
            if (args.Length != 1 || !Uri.TryCreate(args[0], UriKind.Absolute, out Uri uri) ||
                uri.Host != "127.0.0.1" || uri.Scheme != "http" || uri.Port < 1024) {
                MessageBox.Show("ORCA Nova 런처를 통해 실행해 주세요.", "ORCA Nova");
                return;
            }
            Application.Run(new MainWindow(args[0]));
        }
    }

    public sealed class MainWindow : Form {
        // Make the native WinForms frame visually consistent with the dark HTML UI.
        // DWM immersive-dark mode supports Windows 10; custom caption colors require Windows 11.
        [DllImport("dwmapi.dll", ExactSpelling=true)]
        private static extern int DwmSetWindowAttribute(IntPtr hwnd, int attribute, ref int value, int valueSize);

        private const int DWMWA_USE_IMMERSIVE_DARK_MODE = 20;
        private const int DWMWA_USE_IMMERSIVE_DARK_MODE_LEGACY = 19;
        private const int DWMWA_BORDER_COLOR = 34;
        private const int DWMWA_CAPTION_COLOR = 35;
        private const int DWMWA_TEXT_COLOR = 36;

        private static int ToColorRef(Color color) =>
            color.R | (color.G << 8) | (color.B << 16);

        protected override void OnHandleCreated(EventArgs e) {
            base.OnHandleCreated(e);
            ApplyDarkTitleBar();
        }

        protected override void OnShown(EventArgs e) {
            base.OnShown(e);
            ApplyDarkTitleBar();
        }

        private void ApplyDarkTitleBar() {
            if (!IsHandleCreated) return;
            try {
                int enabled = 1;
                int result = DwmSetWindowAttribute(Handle, DWMWA_USE_IMMERSIVE_DARK_MODE,
                    ref enabled, sizeof(int));
                if (result != 0) {
                    // Older Windows 10 releases used attribute 19 instead of 20.
                    DwmSetWindowAttribute(Handle, DWMWA_USE_IMMERSIVE_DARK_MODE_LEGACY,
                        ref enabled, sizeof(int));
                }

                // Windows 11: paint the actual native caption to match the app.
                // Unsupported attributes safely return an HRESULT on Windows 10.
                int caption = ToColorRef(Color.FromArgb(14, 20, 30));
                int textColor = ToColorRef(Color.FromArgb(237, 244, 255));
                int border = ToColorRef(Color.FromArgb(38, 53, 73));
                DwmSetWindowAttribute(Handle, DWMWA_CAPTION_COLOR, ref caption, sizeof(int));
                DwmSetWindowAttribute(Handle, DWMWA_TEXT_COLOR, ref textColor, sizeof(int));
                DwmSetWindowAttribute(Handle, DWMWA_BORDER_COLOR, ref border, sizeof(int));
            } catch (DllNotFoundException) {
                // Still launch ORCA Nova if DWM is unavailable.
            } catch (EntryPointNotFoundException) {
                // Older Windows builds may not provide this API.
            }
        }

        private readonly string localUrl;
        private readonly WebView2 web = new WebView2();
        private bool closingAllowed = false;
        private bool closingInProgress = false;
        private bool cancelClose = false;
        public MainWindow(string url) {
            localUrl = url;
            Text = "ORCA Nova";
            Icon = Icon.ExtractAssociatedIcon(Application.ExecutablePath);
            Width=1460;Height=980;MinimumSize=new Size(1040,700);
            StartPosition=FormStartPosition.CenterScreen;
            BackColor = Color.FromArgb(14,20,30);
            web.Dock=DockStyle.Fill;
            Controls.Add(web);
            Shown += async (_, __) => await OpenView();
        }
        private async Task OpenView() {
            try {
                await web.EnsureCoreWebView2Async(null);
                web.CoreWebView2.WindowCloseRequested += (_, __) => Close();
                web.CoreWebView2.Settings.AreDevToolsEnabled = false;
                web.Source = new Uri(localUrl);
            } catch(Exception ex) {
                MessageBox.Show("WebView2 초기화 오류: " + ex.Message + "\nMicrosoft Edge WebView2 Runtime 설치가 필요할 수 있어요.", "ORCA Nova");
                closingAllowed=true;Close();
            }
        }
        protected override async void OnFormClosing(FormClosingEventArgs e) {
            if(closingAllowed) { base.OnFormClosing(e);return; }
            e.Cancel = true;
            if(closingInProgress)return;
            closingInProgress = true;
            cancelClose=false;
            try {
                using(var client = new HttpClient { Timeout=TimeSpan.FromSeconds(4) }) {
                    var session = new Uri(localUrl).Query.TrimStart('?').Replace("session=", "");
                    var status = new Uri(localUrl).GetLeftPart(UriPartial.Authority) + "/api/status?session=" + Uri.EscapeDataString(session);
                    string body = await client.GetStringAsync(status);
                    // Preserve active ORCA calculation if the user clicks the native window's X.
                    if(body.Contains("\"state\":\"running\"")) {
                        MessageBox.Show("ORCA 계산 중에는 창을 닫을 수 없어요. 계산을 먼저 중단하거나 완료해 주세요.", "ORCA Nova");
                        cancelClose=true;
                        return;
                    }
                    var quit = new Uri(localUrl).GetLeftPart(UriPartial.Authority) + "/api/quit?session=" + Uri.EscapeDataString(session);
                    client.DefaultRequestHeaders.Add("X-ORCA-Session",session);
                    await client.PostAsync(quit, new StringContent("{}", System.Text.Encoding.UTF8, "application/json"));
                }
            } catch { /* allow closing when backend has already stopped during update */ }
            finally { closingInProgress = false; }
            if(!cancelClose && !closingAllowed) {closingAllowed=true;BeginInvoke((Action)(() => Close()));}
        }
    }
}
