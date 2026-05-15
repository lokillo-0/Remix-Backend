package remix_server

import (
	"net/http"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	fortnite "github.com/remixfn/xenon/modules/database/buckets/fortnite"
)

func createExchangeCode(accountID string) (*fortnite.Exchange, error) {
	code := &fortnite.Exchange{
		Bucket:    odin.Bucket{ID: uuid.New().String()},
		Code:      uuid.New().String(),
		AccountID: accountID,
		Created:   time.Now().Format(time.RFC3339),
	}
	if err := odin.Create(code); err != nil {
		return nil, err
	}
	return code, nil
}

func GETMobileLogin(c *gin.Context) {
	state := "mobile:" + uuid.New().String()
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, mobileLoginHTML(state))
}

func mobileSuccessHTML(exchangeCode string) string {
	return `<!DOCTYPE html><html lang="en"><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Remix</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#0d0b1a;min-height:100vh;display:flex;align-items:center;justify-content:center;overflow:hidden}
body::before{content:'';position:fixed;inset:0;background:radial-gradient(ellipse 80% 60% at 20% 60%,rgba(76,29,149,.55) 0%,transparent 70%),radial-gradient(ellipse 60% 50% at 80% 20%,rgba(109,40,217,.35) 0%,transparent 60%),radial-gradient(ellipse 40% 40% at 50% 100%,rgba(55,20,120,.4) 0%,transparent 60%);pointer-events:none}
.card{position:relative;background:rgba(16,13,30,.75);border:1px solid rgba(255,255,255,.07);border-radius:20px;padding:44px 40px 38px;width:380px;text-align:center;backdrop-filter:blur(32px);box-shadow:0 0 0 1px rgba(0,0,0,.4),0 32px 80px rgba(0,0,0,.6),inset 0 1px 0 rgba(255,255,255,.06);animation:up .42s cubic-bezier(.22,1,.36,1)}
@keyframes up{from{opacity:0;transform:translateY(20px)}to{opacity:1;transform:translateY(0)}}
.logo{width:68px;height:68px;object-fit:contain;margin-bottom:20px;display:block;margin-left:auto;margin-right:auto;filter:drop-shadow(0 0 18px rgba(139,92,246,.5))}
h1{color:#f0ecff;font-size:22px;font-weight:700;margin-bottom:6px;letter-spacing:-.02em}
.sub{color:#4a4560;font-size:13px;margin-bottom:28px}
.btn{display:flex;align-items:center;justify-content:center;gap:8px;width:100%;padding:13px;background:linear-gradient(135deg,#7c3aed,#6d28d9);color:#fff;border:none;border-radius:10px;font-size:15px;font-weight:500;cursor:pointer;transition:opacity .15s,transform .1s;letter-spacing:.01em;box-shadow:0 4px 24px rgba(109,40,217,.4)}
.btn:hover{opacity:.88}
.btn:active{transform:scale(.98);opacity:1}
.arr{display:flex;align-items:center;transition:transform .18s cubic-bezier(.22,1,.36,1)}
.btn:hover .arr{transform:translateX(4px)}
</style></head><body>
<div class="card">
<img src="/admin/logo.png" class="logo" alt="Remix">
<h1>Ready to roll.</h1>
<p class="sub">You're all set — head back to the game.</p>
<button class="btn" onclick="window.location.href='com.epicgames.fortnite://fnauth/?code=` + exchangeCode + `'">Continue <span class="arr"><svg width="15" height="15" viewBox="0 0 15 15" fill="none"><path d="M2.5 7.5H12.5M8.5 3.5L12.5 7.5L8.5 11.5" stroke="white" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/></svg></span></button>
</div>
<script>window.location.href="com.epicgames.fortnite://fnauth/?code=` + exchangeCode + `";</script>
</body></html>`
}

func mobileLoginHTML(state string) string {
	return `<!DOCTYPE html><html lang="en"><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Sign in – Remix</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#0d0b1a;min-height:100vh;display:flex;align-items:center;justify-content:center;overflow:hidden}
body::before{content:'';position:fixed;inset:0;background:radial-gradient(ellipse 80% 60% at 20% 60%,rgba(76,29,149,.55) 0%,transparent 70%),radial-gradient(ellipse 60% 50% at 80% 20%,rgba(109,40,217,.35) 0%,transparent 60%),radial-gradient(ellipse 40% 40% at 50% 100%,rgba(55,20,120,.4) 0%,transparent 60%);pointer-events:none}
.card{position:relative;background:rgba(16,13,30,.75);border:1px solid rgba(255,255,255,.07);border-radius:20px;padding:44px 40px 38px;width:380px;text-align:center;backdrop-filter:blur(32px);box-shadow:0 0 0 1px rgba(0,0,0,.4),0 32px 80px rgba(0,0,0,.6),inset 0 1px 0 rgba(255,255,255,.06);animation:up .42s cubic-bezier(.22,1,.36,1)}
@keyframes up{from{opacity:0;transform:translateY(20px)}to{opacity:1;transform:translateY(0)}}
.logo{width:68px;height:68px;object-fit:contain;margin-bottom:20px;display:block;margin-left:auto;margin-right:auto;filter:drop-shadow(0 0 18px rgba(139,92,246,.5))}
h1{color:#f0ecff;font-size:22px;font-weight:700;margin-bottom:6px;letter-spacing:-.02em}
.sub{color:#4a4560;font-size:13px;margin-bottom:28px}
.btn{display:flex;align-items:center;justify-content:center;gap:10px;width:100%;padding:13px;background:#5865F2;color:#fff;border:none;border-radius:10px;font-size:15px;font-weight:500;cursor:pointer;text-decoration:none;transition:opacity .15s,transform .1s;letter-spacing:.01em;box-shadow:0 4px 24px rgba(88,101,242,.4)}
.btn:hover{opacity:.88}
.btn:active{transform:scale(.98)}
.footer{margin-top:20px;color:#2a2640;font-size:12px}
</style></head><body>
<div class="card">
<img src="/admin/logo.png" class="logo" alt="Remix">
<h1>Sign in to Remix</h1>
<p class="sub">Connect with Discord to continue.</p>
<a class="btn" href="/rmx/server/api/v1/discord/auth?state=` + state + `">
<svg width="20" height="15" viewBox="0 0 71 55" fill="white"><path d="M60.1 4.9A58.6 58.6 0 0 0 45.5.4a41 41 0 0 0-1.8 3.7 54.2 54.2 0 0 0-16.3 0A38.2 38.2 0 0 0 25.6.4 58.4 58.4 0 0 0 11 5C1.6 19 -1 32.7.3 46.2a59 59 0 0 0 18 9.1 44.6 44.6 0 0 0 3.9-6.3 38.4 38.4 0 0 1-6.1-2.9l1.5-1.1a42 42 0 0 0 35.8 0l1.5 1.1a38.3 38.3 0 0 1-6.1 2.9 44.4 44.4 0 0 0 3.8 6.3 58.8 58.8 0 0 0 18-9.1C72 30.6 68.4 17 60.1 4.9ZM23.7 38a6.8 6.8 0 0 1-6.4-7.1 6.8 6.8 0 0 1 6.4-7.2 6.8 6.8 0 0 1 6.4 7.2A6.8 6.8 0 0 1 23.7 38Zm23.6 0a6.8 6.8 0 0 1-6.4-7.1 6.8 6.8 0 0 1 6.4-7.2 6.8 6.8 0 0 1 6.4 7.2A6.8 6.8 0 0 1 47.3 38Z"/></svg>
Continue with Discord
</a>
<p class="footer">By signing in you agree to Remix's terms of service.</p>
</div>
</body></html>`
}
